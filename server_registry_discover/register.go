package serverregistrydiscover

import (
	"context"
	"etcd/logger"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
)

// register 模块提供服务注册至 etcd 的能力
var (
	DefaultEtcdConfig = clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}
)

// etcdAdd 以租约模式添加一对kv 至 etcd
func etcdAdd(client *clientv3.Client, lid clientv3.LeaseID, service string, addr string) error {
	em, err := endpoints.NewManager(client, service)
	if err != nil {
		return err
	}
	//return em.AddEndpoint(c.Ctx(), service+"/"+addr, endpoints.Endpoint{Addr: addr})
	return em.AddEndpoint(client.Ctx(), service+"/"+addr, endpoints.Endpoint{Addr: addr}, clientv3.WithLease(lid))
}

// Register 注册一个服务至 etcd
// 注意 Register 将不会 return（如果没有 error 的话）
func Register(service string, addr string, stop chan error) error {
	// 使用默认配置创建一个 etcd client
	cli, err := clientv3.New(DefaultEtcdConfig)
	if err != nil {
		return fmt.Errorf("create etcd client falied: %v", err)
	}
	defer cli.Close()

	// 调用客户端的 Grant 方法创建一个租约，配置 5s 过期
	resp, err := cli.Grant(context.Background(), 5)
	if err != nil {
		return fmt.Errorf("create lease failed: %v", err)
	}
	leaseId := resp.ID
	// 注册服务
	err = etcdAdd(cli, leaseId, service, addr)
	if err != nil {
		return fmt.Errorf("add etcd record failed: %v", err)
	}
	// 设置服务心跳检测
	ch, err := cli.KeepAlive(context.Background(), leaseId)
	if err != nil {
		return fmt.Errorf("set keepalive failed: %v", err)
	}
	log.Printf("[%s] register service ok\n", addr)
	for {
		select {
		case err := <-stop:
			if err != nil {
				logger.Logger.Error(err.Error())
			}
			return err
		case <-cli.Ctx().Done():
			logger.Logger.Info("service closed")
			return nil
		case _, ok := <-ch:
			// 监听租约
			if !ok {
				logger.Logger.Info("keepalive channel closed")
				// 撤销撤销给定的租约。
				_, err := cli.Revoke(context.Background(), leaseId)
				return err
			}
			logger.Logger.Info("Recv reply from service: %s/%s, ttl:%d", service, addr, resp.TTL)
		}

	}
}
