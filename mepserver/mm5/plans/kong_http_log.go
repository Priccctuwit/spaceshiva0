/*
 * Copyright 2021 Huawei Technologies Co., Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package plans

import (
	"context"
	"io/ioutil"
	"mepserver/common/arch/workspace"
	"mepserver/common/models"
	meputil "mepserver/common/util"
	"mepserver/mp1"
	"net/http"
	"net/url"
	"strconv"

	"github.com/apache/servicecomb-service-center/pkg/log"
	es "github.com/olivere/elastic/v7"
)

const startedAt = "started_at"

const esHost = "http://mep-elasticsearch:9200"

var EsClient *es.Client

func init() {
	EsClient = createEsClient()
}

func createEsClient() *es.Client {
	log.Info("Create Es Client")
	esClient, err := es.NewClient(es.SetSniff(false), es.SetURL(esHost))
	if err != nil {
		log.Error("Connect to es fail.", err)
		return EsClient
	}
	log.Info("Connect to es success")

	exists, err := esClient.IndexExists(meputil.KongHttpLogIndex).Do(context.Background())
	if err != nil {
		log.Error("Index exists", err)
		return esClient
	}

	if exists {
		log.Info("Index exists")
	} else {
		mapping := models.GetHttpLogMapping()
		createIndex, err := esClient.CreateIndex(meputil.KongHttpLogIndex).BodyString(mapping).Do(context.Background())
		if err != nil {
			log.Error("CreateIndex fail.", err)
			return esClient
		}
		if !createIndex.Acknowledged {
			log.Error("CreateIndex fail, not Acknowledged.", err)
			return esClient
		}
	}
	return esClient
}

type CreateKongHttpLog struct {
	workspace.TaskBase
	R       *http.Request `json:"r,in"`
	HttpRsp interface{}   `json:"httpRsp,out"`
}

// When call the api through kong api gateway, the kong http-log plugin will send message to this interface.
// The interface will store the data to elasticsearch for search by other api.
func (t *CreateKongHttpLog) OnRequest(data string) workspace.TaskCode {
	log.Info("CreateKongHttpLog")
	msg, err := ioutil.ReadAll(t.R.Body)
	if err != nil {
		log.Error("read failed", nil)
		t.SetFirstErrorCode(meputil.SerErrFailBase, "read request body error")
		return workspace.TaskFinish
	}

	resp, err := EsClient.Index().Index(meputil.KongHttpLogIndex).BodyString(string(msg)).Do(context.Background())
	if err != nil {
		log.Error("Create doc fail.", err)
	}
	t.HttpRsp = resp

	return workspace.TaskFinish
}

type GetKongHttpLog struct {
	workspace.TaskBase
	R       *http.Request `json:"r,in"`
	HttpRsp interface{}   `json:"httpRsp,out"`
}

// The interface is query called times of the 3rd app registered services and mep self capability from elasticsearch.
func (t *GetKongHttpLog) OnRequest(data string) workspace.TaskCode {
	log.Info("GetKongHttpLog")
	// 3rd app services list
	// registered services name list
	serviceNames := getAllServiceNames()
	appList := statisticAppServices(serviceNames)

	// MEP self capability
	mepList := statisticMepServices()

	res := make(map[string]interface{})
	res["appServices"] = appList
	res["mepServices"] = mepList
	t.HttpRsp = res
	return workspace.TaskFinish
}

func statisticMepServices() []interface{} {
	list := make([]interface{}, 0)

	// service register data
	registerMap := make(map[string]interface{})
	registerMap["name"] = "serviceRegister"
	registerMap["desc"] = ""
	registerMap["callTimes"] = statisticRegisterServices()
	list = append(list, registerMap)
	//services["serviceRegister"] = statisticRegisterServices()

	// service discovery data
	discoveryMap := make(map[string]interface{})
	discoveryMap["name"] = "serviceDiscovery"
	discoveryMap["desc"] = ""
	discoveryMap["callTimes"] = statisticDiscoveryServices()
	list = append(list, discoveryMap)
	//services["serviceDiscovery"] = statisticDiscoveryServices()

	return list
}

func statisticDiscoveryServices() interface{} {
	dayCount := make([]int, meputil.WeekDay)
	for i := 0; i < meputil.WeekDay; i++ {
		boolQuery := es.NewBoolQuery()
		serviceNameQuery := es.NewTermsQuery("service.name", "mepserver")
		boolQuery.Filter(serviceNameQuery)

		upstreamUriQuery := es.NewPrefixQuery("upstream_uri.keyword",
			"/mep/mec_service_mgmt/v1/services")
		boolQuery.Filter(upstreamUriQuery)

		requestMethodQuery := es.NewMatchQuery("request.method", "GET")
		boolQuery.Filter(requestMethodQuery)

		timeRangeQuery := getTimeRange(i)
		boolQuery.Filter(timeRangeQuery)

		resp, err := EsClient.Count(meputil.KongHttpLogIndex).Query(boolQuery).Do(context.Background())
		if err != nil {
			dayCount[i] = 0
		} else {
			dayCount[i] = int(resp)
		}
	}
	return dayCount
}

func statisticRegisterServices() interface{} {
	dayCount := make([]int, meputil.WeekDay)
	for i := 0; i < meputil.WeekDay; i++ {
		boolQuery := es.NewBoolQuery()
		serviceNameQuery := es.NewTermsQuery("service.name", "mepserver")
		boolQuery.Filter(serviceNameQuery)

		upstreamUriQuery := es.NewRegexpQuery("upstream_uri.keyword",
			"/mep/mec_service_mgmt/v1/applications/[-A-Za-z0-9]+/services")
		boolQuery.Filter(upstreamUriQuery)

		requestMethodQuery := es.NewMatchQuery("request.method", "POST")
		boolQuery.Filter(requestMethodQuery)

		timeRangeQuery := getTimeRange(i)
		boolQuery.Filter(timeRangeQuery)

		resp, err := EsClient.Count(meputil.KongHttpLogIndex).Query(boolQuery).Do(context.Background())
		if err != nil {
			dayCount[i] = 0
		} else {
			dayCount[i] = int(resp)
		}
	}
	return dayCount
}

func getTimeRange(i int) *es.RangeQuery {
	// range by day
	if i == 0 {
		return es.NewRangeQuery(startedAt).Gte("now/d")
	} else {
		return es.NewRangeQuery(startedAt).Gte("now-" + strconv.Itoa(i) + "d/d").Lt("now-" + strconv.Itoa(
			i-1) + "d/d")
	}
}

func statisticAppServices(names []string) []interface{} {
	list := make([]interface{}, 0)
	for _, serviceName := range names {
		serviceMap := make(map[string]interface{})
		dayCount := make([]int, meputil.WeekDay)
		for i := 0; i < meputil.WeekDay; i++ {
			boolQuery := es.NewBoolQuery()
			serviceNameQuery := es.NewPrefixQuery("service.name.keyword", serviceName)
			boolQuery.Filter(serviceNameQuery)

			timeRangeQuery := getTimeRange(i)
			boolQuery.Filter(timeRangeQuery)

			resp, err := EsClient.Count(meputil.KongHttpLogIndex).Query(boolQuery).Do(context.Background())
			if err != nil {
				dayCount[i] = 0
			} else {
				dayCount[i] = int(resp)
			}
		}
		serviceMap["name"] = serviceName
		serviceMap["desc"] = ""
		serviceMap["callTimes"] = dayCount
		list = append(list, serviceMap)
	}
	return list
}

func getAllServiceNames() []string {
	serviceNames := make([]string, 0)
	findInstancesResponse, err := meputil.FindInstanceByKey(url.Values{})
	if err != nil {
		log.Errorf(nil, "FindInstanceByKey failed.")
		return serviceNames
	}

	_, serviceInfos := mp1.Mp1CvtSrvDiscover(findInstancesResponse)
	if serviceInfos == nil {
		log.Errorf(nil, "Mp1CvtSrvDiscover failed.")
		return serviceNames
	}

	for _, service := range serviceInfos {
		serviceNames = append(serviceNames, service.SerName)
	}
	log.Infof("serviceNames: %s", serviceNames)
	return serviceNames
}
