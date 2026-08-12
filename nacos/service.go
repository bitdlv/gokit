package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func registerServiceInstance(client naming_client.INamingClient, param vo.RegisterInstanceParam) {
	success, err := client.RegisterInstance(param)
	if !success {
		logx.Errorf("RegisterServiceInstance failed!")
		return
	}
	if err != nil {
		logx.Errorf("RegisterServiceInstance failed!" + err.Error())
		return
	}
	logx.Debugf("RegisterServiceInstance,param:%+v,result:%+v \n\n", param, success)
	return
}

// GetService Get service with serviceName, groupName , clusters
func GetService(client naming_client.INamingClient, param vo.GetServiceParam) (service model.Service, err error) {
	service, err = client.GetService(param)
	if err != nil {
		logx.Errorf("GetService failed!" + err.Error())
		return
	}
	logx.Debugf("GetService,param:%+v, result:%+v \n\n", param, service)
	return
}

// SelectInstances only return the instances of healthy=${HealthyOnly},enable=true and weight>0
func SelectInstances(client naming_client.INamingClient, param vo.SelectInstancesParam) (instances []model.Instance, err error) {
	instances, err = client.SelectInstances(param)
	if err != nil {
		panic("SelectInstances failed!" + err.Error())
	}
	logx.Debugf("SelectInstances,param:%+v, result:%+v \n\n", param, instances)
	return
}

// todo 待实现
func subscribe(client naming_client.INamingClient, param *vo.SubscribeParam) {
	err := client.Subscribe(param)
	if err != nil {
		return
	}
}

// todo 待实现
func unSubscribe(client naming_client.INamingClient, param *vo.SubscribeParam) {
	err := client.Unsubscribe(param)
	if err != nil {
		return
	}
}

// GetAllService todo 待实现
func GetAllService(client naming_client.INamingClient, param vo.GetAllServiceInfoParam) {
	service, err := client.GetAllServicesInfo(param)
	if err != nil {
		panic("GetAllService failed!")
	}
	logx.Debugf("GetAllService,param:%+v, result:%+v \n\n", param, service)
}
