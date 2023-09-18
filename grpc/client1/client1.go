package main

import (
	"context"
	"fmt"
	"time"

	"github.com/1055373165/groupcache/conf"
	pb "github.com/1055373165/groupcache/groupcachepb"
	"github.com/1055373165/groupcache/logger"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
)

const etcdUrl = "http://localhost:2379"
const serviceName = "groupcache"

func main() {
	//bd := &ChihuoBuilder{addrs: map[string][]string{"/api": []string{"localhost:8001", "localhost:8002", "localhost:8003"}}}
	//resolver.Register(bd)
	conf.Init()
	etcdClient, err := clientv3.NewFromURL(etcdUrl)
	if err != nil {
		panic(err)
	}
	etcdResolver, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(err)
	}
	conn, err := grpc.Dial(fmt.Sprintf("etcd:///%s", serviceName), grpc.WithResolvers(etcdResolver), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, roundrobin.Name)))

	if err != nil {
		fmt.Printf("err: %v", err)
		return
	}

	ServerClient := pb.NewGroupCacheClient(conn)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		helloRespone, err := ServerClient.Get(ctx, &pb.GetRequest{
			Group: "scores",
			Key:   "李四",
		})
		if err != nil {
			fmt.Printf("err: %v", err)
			return
		}
		logger.Logger.Infof("查询到的分数为：%v🍪\n", helloRespone)
		helloRespone, err = ServerClient.Get(ctx, &pb.GetRequest{
			Group: "scores",
			Key:   "张三",
		})
		if err != nil {
			fmt.Printf("err: %v", err)
			return
		}
		logger.Logger.Infof("查询到的分数为：%v🍪\n", helloRespone)
		helloRespone, err = ServerClient.Get(ctx, &pb.GetRequest{
			Group: "scores",
			Key:   "zhaobudao",
		})

		if err != nil {
			if err.Error() == "rpc error: code = Unknown desc = record not found" {
				logger.Logger.Infof("查询不到name为 'zhaobudao' 的成绩 %v\n", helloRespone)
			} else {
				return
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}
