package nacos

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"syscall"

	"github.com/bitdlv/gokit/amslogx"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	nacoslog "github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

const (
	nsPort      = 8848
	nsHost      = "nacos-headless.nacos"
	nsNameSpace = "3bf5e50b-424a-4011-afe8-2eea9ad3ee24" // 默认 base 环境的 Namespace ID
)

var (
	nsClientInstance *NsClient
	lock             = &sync.Mutex{}
)

type NsClient struct {
	Cc *constant.ClientConfig
	Sc []constant.ServerConfig
}

// GetNsClientInstance 获取nacos client 实例
func GetNsClientInstance() (*NsClient, error) {
	lock.Lock()
	defer lock.Unlock()
	// 设置Logger
	nacoslog.SetLogger(amslogx.New(context.Background()))
	if nsClientInstance != nil {
		return nsClientInstance, nil
	}
	cc, sc, err := newNsClient()
	if err != nil {
		return nil, err
	}
	nsClientInstance = &NsClient{Cc: cc, Sc: sc}
	return nsClientInstance, nil
}

func newNsClient() (cc *constant.ClientConfig, sc []constant.ServerConfig, err error) {
	host := os.Getenv("CONFIG_HOST")
	if host == "" {
		host = nsHost
	}

	configPort := os.Getenv("CONFIG_PORT")
	port, err := strconv.ParseUint(configPort, 10, 64)
	if err != nil {
		port = nsPort
	}

	// 核心改动 1：直接从环境变量获取 Namespace ID，抛弃易碎的 HTTP 映射查询
	namespace := os.Getenv("NACOS_NAMESPACE_ID")
	if namespace == "" {
		namespace = nsNameSpace // 兜底默认值
	}

	// 核心改动 2：从环境变量获取鉴权账号密码（即便服务端暂时关了 Client 鉴权，注入也不会报错）
	nacosUser := os.Getenv("NACOS_USER")
	nacosPass := os.Getenv("NACOS_PASSWORD")

	logx.Infof("host:%v, port:%v, namespace:%v,nacosuser:%s,nacospassword:%s", host, port, namespace, nacosUser, nacosPass)

	sc = []constant.ServerConfig{
		*constant.NewServerConfig(host, port, constant.WithContextPath("")),
	}
	cc = constant.NewClientConfig(
		constant.WithNamespaceId(namespace),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(false),
		constant.WithLogDir(""),
		constant.WithCacheDir(""),
		constant.WithLogLevel("error"), //降低日志级别
		constant.WithUpdateCacheWhenEmpty(true),
		constant.WithUsername(nacosUser),
		constant.WithPassword(nacosPass),
	)
	return cc, sc, nil
}

func LoadNsConfig(dataId string, c interface{}) error {
	nsClientInstance, err := GetNsClientInstance()
	if err != nil {
		fmt.Printf("Failed to create NcosClient: %v\n", err)
	}

	cc, sc := nsClientInstance.Cc, nsClientInstance.Sc
	nsClient, err := clients.NewConfigClient(vo.NacosClientParam{ClientConfig: cc, ServerConfigs: sc})
	if err != nil {
		logx.Errorf("clients.NewConfigClient failed:%v", err)
		return err
	}

	res, err := nsClient.GetConfig(vo.ConfigParam{DataId: dataId, Group: dataId})
	//配置监听
	go listenConfig(nsClient, dataId)
	//注册当前服务
	go registerService(cc, sc)

	return conf.LoadFromYamlBytes([]byte(res), c)
}

func LoadRpcClientConf(serviceName string) zrpc.RpcClientConf {
	cconf := zrpc.RpcClientConf{
		Endpoints:   nil,
		Timeout:     10000,
		Target:      serviceName + ":8080",
		Middlewares: zrpc.ClientMiddlewaresConf{},
	}
	nacosClient, err := GetNsClientInstance()
	if err != nil {
		logx.Errorf("GetNsClientInstance Failed:%v, return default config:%v", err, cconf)
		return cconf
	}
	nameingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  nacosClient.Cc,
			ServerConfigs: nacosClient.Sc,
		},
	)
	if err != nil {
		logx.Errorf("clients.NewNamingClient Failed:%v", err)
		return cconf
	}
	params := vo.SelectInstancesParam{
		Clusters:    []string{"zhiwei"},
		ServiceName: serviceName,
		GroupName:   serviceName,
		HealthyOnly: true,
	}

	// 注意：SelectInstances 的具体实现在你原代码的其他地方，这里保持原样调用
	instances, err := SelectInstances(nameingClient, params)
	if nil != err {
		logx.Errorf("SelectInstances Failed:%v, return default RpcClientConf:%v", err, cconf)
		return cconf
	}
	logx.Infof("SelectInstances result:%+v", instances)
	for _, ins := range instances {
		cconf.Endpoints = append(cconf.Endpoints, ins.Ip+":"+strconv.FormatInt(int64(ins.Port), 10))
	}
	return cconf
}

func registerService(cc *constant.ClientConfig, sc []constant.ServerConfig) {
	// 从环境变量中获取服务名称和 IP
	serviceName := os.Getenv("SERVICE_NAME")
	serviceIP := os.Getenv("SERVICE_CLUSTER_IP")
	servicePort := os.Getenv("SERVICE_PORT")
	if serviceName == "" || serviceIP == "" {
		logx.Infof("Missing required environment variables: SERVICE_NAME, SERVICE_CLUSTER_IP Skipped!!!")
		return
	}
	// 注册服务
	nclient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  cc,
			ServerConfigs: sc,
		},
	)
	if err != nil {
		logx.Errorf("clients.NewNamingClient Failed:%v", err)
		return
	}
	// 暂时支持 RPC Server注册
	if servicePort == "" {
		servicePort = "8080"
	}
	port, _ := strconv.ParseInt(servicePort, 10, 64)

	// 注意：registerServiceInstance 的具体实现在你原代码的其他地方，这里保持原样调用
	registerServiceInstance(nclient, vo.RegisterInstanceParam{
		Ip:          serviceIP,
		Port:        uint64(port),
		ServiceName: serviceName,
		GroupName:   serviceName,
		Weight:      10, Enable: true, Healthy: true, Ephemeral: true,
		Metadata: map[string]string{},
	})
}

func listenConfig(nsClient config_client.IConfigClient, dataId string) {
	err := nsClient.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  dataId,
		OnChange: func(namespace, group, dataId, data string) {
			pid := os.Getpid()
			process, err := os.FindProcess(pid)
			if err != nil {
				logx.Errorf("Failed to find process: %v", err)
				return
			}
			err = process.Signal(syscall.SIGHUP)
			if err != nil {
				logx.Errorf("Failed to restart process: %v", err)
				return
			}
			logx.Info("Service restarted")
		},
	})
	if err != nil {
		fmt.Printf("Failed to listen config changes: %v\n", err)
		os.Exit(1)
	}
}
