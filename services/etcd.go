package services

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/rancher/rke/docker"
	"github.com/rancher/rke/hosts"
	"github.com/rancher/types/apis/cluster.cattle.io/v1"
	"github.com/sirupsen/logrus"
)

func RunEtcdPlane(etcdHosts []hosts.Host, etcdService v1.ETCDService) error {
	logrus.Infof("[%s] Building up Etcd Plane..", ETCDRole)
	initCluster := getEtcdInitialCluster(etcdHosts)
	for _, host := range etcdHosts {
		imageCfg, hostCfg := buildEtcdConfig(host, etcdService, initCluster)
		err := docker.DoRunContainer(host.DClient, imageCfg, hostCfg, EtcdContainerName, host.AdvertisedHostname, ETCDRole)
		if err != nil {
			return err
		}
	}
	logrus.Infof("[%s] Successfully started Etcd Plane..", ETCDRole)
	return nil
}

func RemoveEtcdPlane(etcdHosts []hosts.Host) error {
	logrus.Infof("[%s] Tearing down Etcd Plane..", ETCDRole)
	for _, host := range etcdHosts {
		err := docker.DoRemoveContainer(host.DClient, EtcdContainerName, host.AdvertisedHostname)
		if err != nil {
			return err
		}
	}
	logrus.Infof("[%s] Successfully teared down Etcd Plane..", ETCDRole)
	return nil
}

func buildEtcdConfig(host hosts.Host, etcdService v1.ETCDService, initCluster string) (*container.Config, *container.HostConfig) {
	imageCfg := &container.Config{
		Image: etcdService.Image,
		Cmd: []string{"/usr/local/bin/etcd",
			"--name=etcd-" + host.AdvertisedHostname,
			"--data-dir=/etcd-data",
			"--advertise-client-urls=http://" + host.AdvertiseAddress + ":2379,http://" + host.AdvertiseAddress + ":4001",
			"--listen-client-urls=http://0.0.0.0:2379",
			"--initial-advertise-peer-urls=http://" + host.AdvertiseAddress + ":2380",
			"--listen-peer-urls=http://0.0.0.0:2380",
			"--initial-cluster-token=etcd-cluster-1",
			"--initial-cluster=" + initCluster,
			"--initial-cluster-state=new"},
	}
	hostCfg := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "always"},
		Binds: []string{
			"/var/lib/etcd:/etcd-data"},
		PortBindings: nat.PortMap{
			"2379/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "2379",
				},
			},
			"2380/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "2380",
				},
			},
		},
	}
	for arg, value := range etcdService.ExtraArgs {
		cmd := fmt.Sprintf("--%s=%s", arg, value)
		imageCfg.Cmd = append(imageCfg.Cmd, cmd)
	}

	return imageCfg, hostCfg
}

func GetEtcdConnString(hosts []hosts.Host) string {
	connString := ""
	for i, host := range hosts {
		connString += "http://" + host.AdvertiseAddress + ":2379"
		if i < (len(hosts) - 1) {
			connString += ","
		}
	}
	return connString
}

func getEtcdInitialCluster(hosts []hosts.Host) string {
	initialCluster := ""
	for i, host := range hosts {
		initialCluster += fmt.Sprintf("etcd-%s=http://%s:2380", host.AdvertisedHostname, host.AdvertiseAddress)
		if i < (len(hosts) - 1) {
			initialCluster += ","
		}
	}
	return initialCluster
}
