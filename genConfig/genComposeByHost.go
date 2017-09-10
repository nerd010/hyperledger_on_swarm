package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

//Main 给三台机器部署fabric集群生成配置文件每个org不能超过9个peer，org不能超过6个,三个zookeeper，三个kafka，两个order都采用host模式部署，两个ca也采用host模式部署
func Main(peers, orgs, zks, kafkas, orderers int, net, domain string, hosts []string) {
	for i := 0; i < zks; i++ {
		zkdc := genZkService(i, zks, net, domain, hosts)
		filename := "zookeeper" + strconv.Itoa(i) + "." + domain + ".yaml"
		os.RemoveAll(filename)
		f, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		data, _ := yaml.Marshal(zkdc)
		f.Write(data)
		f.Close()
	}

	for i := 0; i < kafkas; i++ {
		kafdc := genKafkaService(i, kafkas, net, domain, hosts)
		filename := "kafka" + strconv.Itoa(i) + "." + domain + ".yaml"
		os.RemoveAll(filename)
		f, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		data, _ := yaml.Marshal(kafdc)
		f.Write(data)
		f.Close()
	}

	dcs := GenPeersWithCouchDb(peers, orgs, hosts, net, domain)
	for name, dc := range dcs {
		os.RemoveAll(name + ".yaml")
		f, err := os.Create(name + ".yaml")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		data, _ := yaml.Marshal(dc)
		f.Write(data)
		f.Close()
	}

	for i := 1; i <= orgs; i++ {
		cadc := genCaService(i, domain, net)
		filename := "ca" + strconv.Itoa(i) + "." + domain + ".yaml"
		os.RemoveAll(filename)
		f, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		data, _ := yaml.Marshal(cadc)
		f.Write(data)
		f.Close()
	}

	for i := 0; i < orderers; i++ {
		dc := genOrderers(i, net, domain, nil)
		filename := "orderer" + strconv.Itoa(i) + "." + domain + ".yaml"
		os.RemoveAll(filename)
		f, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		data, _ := yaml.Marshal(dc)
		f.Write(data)
		f.Close()
	}
	dc := genCliService(peers, orgs, net, domain, nil)
	filename := "cli.yaml"
	os.RemoveAll(filename)
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	data, _ := yaml.Marshal(dc)

	f.Write(data)
	f.Close()

}

//GenPeersWithCouchDb 生成带couchdb的peer节点配置信息，每个配置单独生成一个文件 在一台主机上执行
func GenPeersWithCouchDb(peerNum, orgNum int, hosts []string, network string, domain string) map[string]DockerCompose {
	dcs := make(map[string]DockerCompose)
	for peer := 0; peer < peerNum; peer++ {
		dc, name := genPeersWithCouchDb(peer, orgNum, hosts, network, domain)
		dcs[name] = dc
	}
	return dcs
}

func genPeersWithCouchDb(peer, orgNum int, hosts []string, network string, domain string) (DockerCompose, string) {
	var name = "peer" + strconv.Itoa(peer)
	services := make(map[string]*Service)
	for i := 1; i < orgNum; i++ {
		service := genPeersWithCouchDbService(peer, orgNum, hosts, network, domain)
		for k, v := range service {
			services[k] = v
			if strings.Contains(k, "couchdb") {
				continue
			}
			name = k
		}

	}
	dc := DockerCompose{
		Version:  "3",
		Services: services,
		Networks: make(map[string]*Network),
	}

	networks := make(map[string]*Network)
	networks[network] = &Network{
		External: &External{
			Name: network,
		},
	}
	dc.Networks = networks

	return dc, name
}
func genCliService(peerNum, orgNum int, net, domain string, hosts []string) DockerCompose {

	dc := DockerCompose{
		Version:  "3",
		Services: make(map[string]*Service),
		Networks: make(map[string]*Network),
	}
	networks := make(map[string]*Network)
	networks[net] = &Network{
		External: &External{
			Name: net,
		},
	}
	dc.Networks = networks
	service := Service{
		Image:    "hyperledger/fabric-tools" + TAG,
		Hostname: "cli",
	}
	service.Networks = make(map[string]*ServNet, 1)
	service.Networks[net] = &ServNet{
		Aliases: []string{"cli"},
	}
	service.Environment = make([]string, 13)
	service.Environment[0] = "CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=" + net
	service.Environment[1] = "GOPATH=/opt/gopath"
	service.Environment[2] = "CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock"
	service.Environment[3] = "CORE_LOGGING_LEVEL=DEBUG"
	service.Environment[4] = "CORE_PEER_ID=cli"
	service.Environment[5] = "CORE_PEER_ADDRESS=peer0.org1." + domain + ":7051"
	service.Environment[6] = "CORE_PEER_LOCALMSPID=Org1MSP"
	service.Environment[7] = "CORE_PEER_TLS_ENABLED=true"
	service.Environment[8] = "CORE_PEER_TLS_CERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1." + domain + "/peers/peer0.org1." + domain + "/tls/server.crt"
	service.Environment[9] = "CORE_PEER_TLS_KEY_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1." + domain + "/peers/peer0.org1." + domain + "/tls/server.key"
	service.Environment[10] = "CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1." + domain + "/peers/peer0.org1." + domain + "/tls/ca.crt"
	service.Environment[11] = "CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1." + domain + "/users/Admin@org1." + domain + "/msp"
	service.Environment[12] = "GODEBUG=netdns=go" //采用纯go的dns解析，cgo的会有panic
	service.WorkingDir = "/opt/gopath/src/github.com/hyperledger/fabric/peer"
	service.Command = "sleep 36000000000000"
	service.Volumes = make([]string, 5)
	service.Volumes[0] = "/var/run/:/host/var/run/"
	service.Volumes[1] = "./../chaincode/:/opt/gopath/src/github.com/hyperledger/fabric/examples/chaincode/go"
	service.Volumes[2] = "./crypto-config:/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/"
	service.Volumes[3] = "./scripts:/opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/"
	service.Volumes[4] = "./channel-artifacts:/opt/gopath/src/github.com/hyperledger/fabric/peer/channel-artifacts"
	service.ExtraHosts = make([]string, 0)
	service.ExtraHosts = hosts
	return dc

}
func genPeersWithCouchDbService(peerIndex, orgNum int, hosts []string, net string, domain string) map[string]*Service {
	m := make(map[string]*Service)

	for i := 1; i <= orgNum; i++ {
		tag := "peer" + strconv.Itoa(peerIndex) + "Org" + strconv.Itoa(i)
		s := genCouchDbService(tag, net)

		m[s.Hostname] = s
		hostname := fmt.Sprintf("peer%d.org%d.%s", peerIndex, i, domain)
		peerService := Service{
			Hostname: hostname,
			Image:    "hyperledger/fabric-peer" + TAG,
		}
		peerService.Networks = make(map[string]*ServNet)
		peerService.Networks[net] = &ServNet{
			Aliases: []string{hostname},
		}
		port7051 := strconv.Itoa((i-1)*10000 + 7051 + peerIndex*100)
		org1stPort7051 := strconv.Itoa((i-1)*10000 + 7051)
		port7053 := strconv.Itoa((i-1)*10000 + 7053 + peerIndex*100)
		peerService.Environment = make([]string, 18)
		peerService.Environment[0] = "CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock"
		peerService.Environment[1] = "CORE_LOGGING_LEVEL=DEBUG"
		peerService.Environment[2] = "CORE_PEER_TLS_ENABLED=true"
		peerService.Environment[3] = "CORE_PEER_GOSSIP_USELEADERELECTION=true"
		peerService.Environment[4] = "CORE_PEER_GOSSIP_ORGLEADER=false"
		peerService.Environment[5] = "CORE_PEER_PROFILE_ENABLED=true"
		peerService.Environment[6] = "CORE_PEER_TLS_CERT_FILE=/etc/hyperledger/fabric/tls/server.crt"
		peerService.Environment[7] = "CORE_PEER_TLS_KEY_FILE=/etc/hyperledger/fabric/tls/server.key"
		peerService.Environment[8] = "CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt"
		peerService.Environment[9] = "CORE_PEER_ID=" + hostname
		peerService.Environment[10] = "CORE_PEER_ADDRESS=" + hostname + ":" + port7051
		peerService.Environment[11] = "CORE_PEER_GOSSIP_EXTERNALENDPOINT=" + hostname + ":" + port7051
		peerService.Environment[12] = "CORE_PEER_LOCALMSPID=Org" + strconv.Itoa(i) + "MSP"
		peerService.Environment[13] = "CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=" + net
		peerService.Environment[14] = "CORE_LEDGER_STATE_STATEDATABASE=CouchDB"
		peerService.Environment[15] = "CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS=couchdb" + tag + ":5984"
		peerService.Environment[16] = "CORE_PEER_GOSSIP_BOOTSTRAP=peer0.org" + strconv.Itoa(i) + "." + domain + ":" + org1stPort7051
		peerService.Environment[17] = "GODEBUG=netdns=go" //采用纯go的dns解析，cgo的会有panic
		//peerService.Environment[3]  = "CORE_PEER_ENDORSER_ENABLED=true"
		//peerService.Environment[6]  = "CORE_PEER_GOSSIP_SKIPHANDSHAKE=true"
		peerService.WorkingDir = "/opt/gopath/src/github.com/hyperledger/fabric/peer"
		peerService.Command = "peer node start"
		peerService.Volumes = make([]string, 3)
		peerService.Volumes[0] = "/var/run/:/host/var/run/"
		peerService.Volumes[1] = "./crypto-config/peerOrganizations/org" + strconv.Itoa(i) + "." + domain + "/peers/" + hostname + "/msp:/etc/hyperledger/fabric/msp"
		peerService.Volumes[2] = "./crypto-config/peerOrganizations/org" + strconv.Itoa(i) + "." + domain + "/peers/" + hostname + "/tls:/etc/hyperledger/fabric/tls"
		peerService.Depends = make([]string, 1)
		peerService.Depends[0] = s.Hostname
		peerService.ExtraHosts = make([]string, 0)
		peerService.ExtraHosts = hosts
		peerService.Ports = make([]string, 2)
		peerService.Ports[0] = port7051 + ":7051"
		peerService.Ports[1] = port7053 + ":7053"
		peerService.ExtraHosts = make([]string, len(hosts))
		peerService.ExtraHosts = hosts

		m[hostname] = &peerService
	}

	return m
}

func genCouchDbService(tag string, net string) *Service {
	name := "couchdb" + tag
	s := Service{
		Hostname: name,
		Image:    "hyperledger/fabric-couchdb" + TAG,
		Networks: make(map[string]*ServNet, 1),
	}
	s.Networks[net] = &ServNet{
		Aliases: []string{name},
	}
	return &s
}

func genOrdererPorts(seq int) []string {
	p := strconv.Itoa(seq*10000 + 7050)
	ports := make([]string, 0)
	ports = append(ports, p+":7050")
	return ports
}

func genPeersPorts(peerNum, orgNum int) []string {
	ports := make([]string, 0)
	p := strconv.Itoa(orgNum*10000 + 7051 + peerNum*100)
	p2 := strconv.Itoa(orgNum*10000 + 7053 + peerNum*100)
	ports = append(ports, p, p2)
	return ports
}

func genZkService(index, total int, net, domain string, hosts []string) DockerCompose {
	dc := DockerCompose{
		Version: "3",
	}
	hostname := "zookeeper" + strconv.Itoa(index) + "." + domain

	var zookeeperArray []string
	for i := 0; i < total; i++ {
		zookeeperArray = append(zookeeperArray, "server."+strconv.Itoa(i+1)+"=zookeeper"+strconv.Itoa(i)+"."+domain+":2888:3888")
	}

	zlist := strings.Join(zookeeperArray, " ")
	service := &Service{
		Hostname: hostname,
	}
	service.Networks = make(map[string]*ServNet, 1)
	service.Networks[net] = &ServNet{
		Aliases: []string{hostname},
	}
	service.Image = "hyperledger/fabric-zookeeper" + TAG
	service.Environment = make([]string, 3)
	service.Environment[0] = "CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=" + net
	service.Environment[1] = "ZOO_MY_ID=" + strconv.Itoa(index+1)
	service.Environment[2] = "ZOO_SERVERS=" + zlist

	service.Ports = make([]string, 3)
	service.Ports[0] = "2888:2888"
	service.Ports[1] = "3888:3888"
	service.Ports[2] = "2181:2181"
	service.ExtraHosts = make([]string, len(hosts))
	service.ExtraHosts = hosts
	service.NetworkMode = "host"
	dc.Services = make(map[string]*Service)
	dc.Services[hostname] = service

	dc.Networks = make(map[string]*Network)
	networks := make(map[string]*Network)
	networks[net] = &Network{
		External: &External{
			Name: net,
		},
	}
	dc.Networks = networks
	return dc
}

func genKafkaService(index, total int, net, domain string, hosts []string) DockerCompose {

	dc := DockerCompose{
		Version: "3",
	}
	hostname := "kafka" + strconv.Itoa(index)
	service := &Service{
		Hostname: hostname + "." + domain,
	}
	service.Image = "hyperledger/fabric-kafka" + TAG
	service.Environment = make([]string, 8)
	service.Environment[0] = "CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=" + net
	service.Environment[1] = "KAFKA_MESSAGE_MAX_BYTES=103809024"       // 99 MB
	service.Environment[2] = "KAFKA_REPLICA_FETCH_MAX_BYTES=103809024" // 99 MB
	service.Environment[3] = "KAFKA_UNCLEAN_LEADER_ELECTION_ENABLE=false"
	service.Environment[4] = "KAFKA_DEFAULT_REPLICATION_FACTOR=3"
	service.Environment[5] = "KAFKA_MIN_INSYNC_REPLICAS=2"

	var zkarray []string
	for i := 0; i < total; i++ {
		zkarray = append(zkarray, "zookeeper"+strconv.Itoa(i)+"."+domain+":2181")
	}
	zookeeperString := strings.Join(zkarray, ",")
	service.Environment[6] = "KAFKA_ZOOKEEPER_CONNECT=" + zookeeperString
	service.Environment[7] = "KAFKA_BROKER_ID=" + strconv.Itoa(index)
	service.Ports = make([]string, 2)
	service.Ports[0] = "9092:9092"
	service.Ports[1] = "9093:9093"

	service.ExtraHosts = make([]string, len(hosts))
	service.ExtraHosts = hosts
	service.NetworkMode = "host"

	dc.Services = make(map[string]*Service)
	dc.Services[service.Hostname] = service
	dc.Networks = make(map[string]*Network)
	networks := make(map[string]*Network)
	networks[net] = &Network{
		External: &External{
			Name: net,
		},
	}
	dc.Networks = networks

	return dc
}

func genOrderers(index int, net, domain string, hosts []string) DockerCompose {
	dc := DockerCompose{
		Version: "3",
	}

	hostname := "orderer" + strconv.Itoa(index) + "." + domain
	service := &Service{
		Hostname: hostname,
	}

	service.Image = "hyperledger/fabric-orderer" + TAG
	service.Environment = make([]string, 15)
	service.Environment[0] = "CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=" + net
	service.Environment[1] = "ORDERER_GENERAL_LOGLEVEL=debug"
	service.Environment[2] = "ORDERER_GENERAL_LISTENADDRESS=0.0.0.0"
	service.Environment[3] = "ORDERER_GENERAL_GENESISMETHOD=file"
	service.Environment[4] = "ORDERER_GENERAL_GENESISFILE=/var/hyperledger/orderer/orderer.genesis.block"
	service.Environment[5] = "ORDERER_GENERAL_LOCALMSPID=OrdererMSP"
	service.Environment[6] = "ORDERER_GENERAL_LOCALMSPDIR=/var/hyperledger/orderer/msp"
	service.Environment[7] = "ORDERER_GENERAL_TLS_ENABLED=true"
	service.Environment[8] = "ORDERER_GENERAL_TLS_PRIVATEKEY=/var/hyperledger/orderer/tls/server.key"
	service.Environment[9] = "ORDERER_GENERAL_TLS_CERTIFICATE=/var/hyperledger/orderer/tls/server.crt"
	service.Environment[10] = "ORDERER_GENERAL_TLS_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]"
	service.Environment[11] = "ORDERER_KAFKA_RETRY_SHORTINTERVAL=1s"
	service.Environment[12] = "ORDERER_KAFAK_RETRY_SHORTTOTAL=30s"
	service.Environment[13] = "ORDERER_KAFKA_VERBOSE=true"
	service.Environment[14] = "GODEBUG=netdns=go" //采用纯go的dns解析，cgo的会有panic

	service.WorkingDir = "/opt/gopath/src/github.com/hyperledger/fabric"
	service.Command = "orderer"

	service.Volumes = make([]string, 3)
	service.Volumes[0] = "./channel-artifacts/genesis.block:/var/hyperledger/orderer/orderer.genesis.block"
	service.Volumes[1] = "./crypto-config/ordererOrganizations/" + domain + "/orderers/" + hostname + "/msp:/var/hyperledger/orderer/msp"
	service.Volumes[2] = "./crypto-config/ordererOrganizations/" + domain + "/orderers/" + hostname + "/tls/:/var/hyperledger/orderer/tls"
	service.Ports = make([]string, 1)
	service.Ports[0] = "7050:7050"
	service.ExtraHosts = make([]string, len(hosts))
	service.ExtraHosts = hosts
	dc.Services = make(map[string]*Service)
	dc.Services[service.Hostname] = service
	dc.Networks = make(map[string]*Network)
	networks := make(map[string]*Network)
	networks[net] = &Network{
		External: &External{
			Name: net,
		},
	}
	dc.Networks = networks
	return dc
}

func genCaService(org int, domain, net string) DockerCompose {
	dc := DockerCompose{
		Version: "3",
	}
	hostname := "ca.org" + strconv.Itoa(org) + "." + domain
	service := &Service{
		Hostname: hostname,
	}
	service.Networks = make(map[string]*ServNet, 1)
	service.Networks[net] = &ServNet{
		Aliases: []string{hostname},
	}
	orgId := strconv.Itoa(org)
	service.Image = "hyperledger/fabric-ca" + TAG
	service.Environment = make([]string, 5)
	service.Environment[0] = "FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server"
	service.Environment[1] = "FABRIC_CA_SERVER_CA_NAME=" + hostname
	service.Environment[2] = "FABRIC_CA_SERVER_TLS_ENABLED=true"
	service.Environment[3] = "FABRIC_CA_SERVER_TLS_CERTFILE=/etc/hyperledger/fabric-ca-server-config/" + hostname + "-cert.pem"
	service.Environment[4] = "FABRIC_CA_SERVER_TLS_KEYFILE=/etc/hyperledger/fabric-ca-server-config/CA" + orgId + "_PRIVATE_KEY"
	service.Command = "sh -c 'fabric-ca-server start --ca.certfile /etc/hyperledger/fabric-ca-server-config/" + hostname + "-cert.pem --ca.keyfile /etc/hyperledger/fabric-ca-server-config/CA" + orgId + "_PRIVATE_KEY -b admin:adminpw -d'"
	service.Volumes = make([]string, 1)
	service.Volumes[0] = "./crypto-config/peerOrganizations/org" + orgId + "." + domain + "/ca/:/etc/hyperledger/fabric-ca-server-config"
	dc.Services = make(map[string]*Service)
	dc.Services[service.Hostname] = service
	dc.Networks = make(map[string]*Network)
	networks := make(map[string]*Network)
	networks[net] = &Network{
		External: &External{
			Name: net,
		},
	}
	dc.Networks = networks
	return dc
}
