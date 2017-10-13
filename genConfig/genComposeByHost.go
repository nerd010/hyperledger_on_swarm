package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/baoyangc/yaml"
)

//Main 给三台机器部署fabric集群生成配置文件每个org不能超过9个peer，org不能超过6个,三个zookeeper，三个kafka，两个order都采用host模式部署，两个ca也采用host模式部署
func Main(peers, orgs, zks, kafkas, orderers int, net, domain string, hosts []string, addr string, prod bool) {
	for i := 0; i < zks; i++ {
		zkdc := genZkService(i, zks, net, domain, hosts, addr)
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
		kafdc := genKafkaService(i, zks, net, domain, hosts, baseAddr, prod)
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

	dcs := GenPeersWithCouchDb(peers, orgs, hosts, net, domain, addr, prod)
	for name, dc := range dcs {
		filename := name + "." + domain + ".yaml"
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

	for i := 1; i <= orgs; i++ {
		cadc := genCaService(i, domain, net, baseAddr)
		filename := "ca.org" + strconv.Itoa(i) + "." + domain + ".yaml"
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
		dc := genOrderers(i, net, domain, baseAddr, nil, prod)
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
	dc := genCliService(peers, orgs, net, domain, nil, prod)
	filename := "cli_" + domain + ".yaml"
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

func GenPeersWithCouchDb(peerNum, orgNum int, hosts []string, network string, domain, addr string, prod bool) map[string]DockerCompose {
	result := make(map[string]DockerCompose)
	for peer := 0; peer < peerNum; peer++ {

		dcs := genPeersWithCouchDb(peer, orgNum, hosts, network, domain, addr, prod)
		for k, v := range dcs {
			result[k] = v
		}

	}
	return result
}

//GenPeersWithCouchDb 生成配置
func genPeersWithCouchDb(peerIndex, orgNum int, hosts []string, network, domain, addr string, prod bool) map[string]DockerCompose {
	dcs := make(map[string]DockerCompose)
	for i := 1; i <= orgNum; i++ {
		m := genPeersWithCouchDbService(peerIndex, i, hosts, network, domain, addr, prod)

		for k, v := range m {
			if strings.Contains(k, "couchdb") {
				continue
			}
			dcs[k] = v
		}

	}

	return dcs
}
func genCliService(peerNum, orgNum int, net, domain string, hosts []string, prod bool) DockerCompose {

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
		Dns:      make([]string, 1),
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
	service.Dns[0] = baseAddr
	dc.Services["cli"] = &service
	return dc

}
func genPeersWithCouchDbService(peerIndex, orgIndex int, hosts []string, net string, domain string, addr string, prod bool) map[string]DockerCompose {
	result := make(map[string]DockerCompose)

	m := make(map[string]*Service)
	tag := "peer" + strconv.Itoa(peerIndex) + "Org" + strconv.Itoa(orgIndex)
	s := genCouchDbService(tag, net)

	m[s.Hostname] = s
	hostname := fmt.Sprintf("peer%d.org%d.%s", peerIndex, orgIndex, domain)
	peerService := Service{
		Hostname: hostname,
		Image:    "hyperledger/fabric-peer" + TAG,
	}
	peerService.Networks = make(map[string]*ServNet)
	peerService.Networks[net] = &ServNet{
		Aliases: []string{hostname},
	}

	peerService.Environment = make([]string, 20)
	peerService.Environment[0] = "CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock"
	peerService.Environment[1] = "CORE_LOGGING_LEVEL=DEBUG"
	peerService.Environment[5] = "CORE_PEER_PROFILE_ENABLED=true"
	if prod {
		peerService.Environment[1] = "CORE_LOGGING_PEER=warning"
		peerService.Environment[5] = "CORE_PEER_PROFILE_ENABLED=false"
	}
	peerService.Environment[2] = "CORE_PEER_TLS_ENABLED=true"
	peerService.Environment[3] = "CORE_PEER_GOSSIP_USELEADERELECTION=true"
	peerService.Environment[4] = "CORE_PEER_GOSSIP_ORGLEADER=false"
	peerService.Environment[6] = "CORE_PEER_TLS_CERT_FILE=/etc/hyperledger/fabric/tls/server.crt"
	peerService.Environment[7] = "CORE_PEER_TLS_KEY_FILE=/etc/hyperledger/fabric/tls/server.key"
	peerService.Environment[8] = "CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt"
	peerService.Environment[9] = "CORE_PEER_ID=" + hostname
	peerService.Environment[10] = "CORE_PEER_ADDRESS=" + hostname + ":7051"
	peerService.Environment[11] = "CORE_PEER_GOSSIP_EXTERNALENDPOINT=" + hostname + ":7051"
	peerService.Environment[12] = "CORE_PEER_LOCALMSPID=Org" + strconv.Itoa(orgIndex) + "MSP"
	peerService.Environment[13] = "CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=" + net
	peerService.Environment[14] = "CORE_LEDGER_STATE_STATEDATABASE=CouchDB"
	peerService.Environment[15] = "CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS=couchdb" + tag + ":5984"
	peerService.Environment[16] = "CORE_PEER_GOSSIP_BOOTSTRAP=peer0.org" + strconv.Itoa(orgIndex) + "." + domain + ":7051"
	peerService.Environment[17] = "GODEBUG=netdns=go" //采用纯go的dns解析，cgo的会有panic
	peerService.Environment[18] = "CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME=admin"
	peerService.Environment[19] = "CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD=U1T6UafF"
	//peerService.Environment[3]  = "CORE_PEER_ENDORSER_ENABLED=true"
	//peerService.Environment[6]  = "CORE_PEER_GOSSIP_SKIPHANDSHAKE=true"
	peerService.WorkingDir = "/opt/gopath/src/github.com/hyperledger/fabric/peer"
	peerService.Command = "peer node start"
	peerService.Volumes = make([]string, 4)
	peerService.Volumes[0] = "/var/run/:/host/var/run/"
	peerService.Volumes[1] = "./crypto-config/peerOrganizations/org" + strconv.Itoa(orgIndex) + "." + domain + "/peers/" + hostname + "/msp:/etc/hyperledger/fabric/msp"
	peerService.Volumes[2] = "./crypto-config/peerOrganizations/org" + strconv.Itoa(orgIndex) + "." + domain + "/peers/" + hostname + "/tls:/etc/hyperledger/fabric/tls"
	peerService.Volumes[3] = "/data/peer/" + hostname + ":/var/hyperledger/production"
	peerService.Privileged = true
	peerService.Depends = make([]string, 1)
	peerService.Depends[0] = s.Hostname
	peerService.Ports = make([]string, 2)
	peerService.Ports[0] = "7051:7051"
	peerService.Ports[1] = "7053:7053"
	peerService.Dns = make([]string, 1)
	peerService.Dns[0] = addr
	peerService.Restart = "on-falure"

	m[tag] = &peerService

	dc := DockerCompose{
		Version:  "3",
		Services: m,
	}
	dc.Networks = make(map[string]*Network)
	dc.Networks[net] = &Network{
		External: &External{
			Name: net,
		},
	}

	result[tag] = dc

	return result
}

func genCouchDbService(tag string, net string) *Service {
	name := "couchdb" + tag
	s := Service{
		Hostname: name,
		Image:    "hyperledger/fabric-couchdb" + TAG,
		Networks: make(map[string]*ServNet, 1),
	}
	s.Environment = make([]string, 2)
	s.Environment[0] = "COUCHDB_USER=admin"
	s.Environment[1] = "COUCHDB_PASSWORD=U1T6UafF"
	s.Restart = "unless-stopped"
	s.User = "root"
	s.Volumes = make([]string, 1)
	s.Volumes[0] = "/data/couchdb/" + name + ":/opt/couchdb/data"
	s.Privileged = true
	s.Networks[net] = &ServNet{
		Aliases: []string{name},
	}
	return &s
}

func genZkService(index, total int, net, domain string, hosts []string, addr string) DockerCompose {
	dc := DockerCompose{
		Version: "3",
	}
	serviceName := "zookeeper" + strconv.Itoa(index)
	hostname := "zookeeper" + strconv.Itoa(index) + "." + domain

	var zookeeperArray []string
	for i := 0; i < total; i++ {
		zookeeperArray = append(zookeeperArray, "server."+strconv.Itoa(i+1)+"=zookeeper"+strconv.Itoa(i)+"."+domain+":2888:3888")
	}

	zlist := strings.Join(zookeeperArray, " ")
	service := &Service{
		Hostname: hostname,
		Dns:      make([]string, 1),
	}

	service.Image = "hyperledger/fabric-zookeeper" + TAG
	service.Environment = make([]string, 3)
	service.Environment[0] = "CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=" + net
	service.Environment[1] = "ZOO_MY_ID=" + strconv.Itoa(index+1)
	service.Environment[2] = "ZOO_SERVERS=" + zlist
	service.Volumes = make([]string, 2)
	service.Volumes[0] = "/data/zookeeper/" + hostname + "/data:/data"
	service.Volumes[1] = "/data/zookeeper/" + hostname + "/datalog:/datalog"

	service.Ports = make([]string, 3)
	service.Ports[0] = "2888:2888"
	service.Ports[1] = "3888:3888"
	service.Ports[2] = "2181:2181"
	service.NetworkMode = "host"
	service.Dns[0] = addr
	service.Restart = "always"
	dc.Networks = make(map[string]*Network)
	networks := make(map[string]*Network)
	networks[net] = &Network{
		External: &External{
			Name: net,
		},
	}
	dc.Networks = networks
	dc.Services = make(map[string]*Service)
	dc.Services[serviceName] = service
	return dc
}

func genKafkaService(index, total int, net, domain string, hosts []string, ns string, prod bool) DockerCompose {

	dc := DockerCompose{
		Version: "3",
	}
	serviceName := "kafka" + strconv.Itoa(index)
	hostname := "kafka" + strconv.Itoa(index) + "." + domain
	service := &Service{
		Hostname: hostname,
		Dns:      make([]string, 1),
	}
	service.Dns[0] = ns
	service.Image = "hyperledger/fabric-kafka" + TAG
	service.Environment = make([]string, 8)
	service.Environment[0] = "CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=" + net
	service.Environment[1] = "KAFKA_MESSAGE_MAX_BYTES=103809024"       // 99 MB
	service.Environment[2] = "KAFKA_REPLICA_FETCH_MAX_BYTES=103809024" // 99 MB
	service.Environment[3] = "KAFKA_UNCLEAN_LEADER_ELECTION_ENABLE=false"
	service.Environment[4] = "KAFKA_DEFAULT_REPLICATION_FACTOR=3"
	service.Environment[5] = "KAFKA_MIN_INSYNC_REPLICAS=2"
	service.Restart = "on-failure"
	var zkarray []string
	for i := 0; i < total; i++ {
		zkarray = append(zkarray, "zookeeper"+strconv.Itoa(i)+"."+domain+":2181")
	}
	zookeeperString := strings.Join(zkarray, ",")
	service.Environment[6] = "KAFKA_ZOOKEEPER_CONNECT=" + zookeeperString
	service.Environment[7] = "KAFKA_BROKER_ID=" + strconv.Itoa(index)
	service.Volumes = make([]string, 1)
	service.Volumes[0] = "/data/kafka/" + hostname + ":/tmp/kafka-logs"
	service.Privileged = true
	service.Ports = make([]string, 2)
	service.Ports[0] = "9092:9092"
	service.Ports[1] = "9093:9093"

	service.NetworkMode = "host"
	dc.Networks = make(map[string]*Network)
	networks := make(map[string]*Network)
	networks[net] = &Network{
		External: &External{
			Name: net,
		},
	}
	dc.Networks = networks
	dc.Services = make(map[string]*Service)
	dc.Services[serviceName] = service

	return dc
}

func genOrderers(index int, net, domain, ns string, hosts []string, prod bool) DockerCompose {
	dc := DockerCompose{
		Version: "3",
	}
	serviceName := "orderer" + strconv.Itoa(index)
	hostname := "orderer" + strconv.Itoa(index) + "." + domain
	service := &Service{
		Hostname: hostname,
		Dns:      make([]string, 1),
	}
	service.Dns[0] = ns

	service.Image = "hyperledger/fabric-orderer" + TAG
	service.Environment = make([]string, 15)
	service.Environment[0] = "CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=" + net
	service.Environment[1] = "ORDERER_GENERAL_LOGLEVEL=debug"
	if prod {
		service.Environment[1] = "ORDERER_GENERAL_LOGLEVEL=warning"
	}
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
	service.Restart = "unless-stopped"
	service.Volumes = make([]string, 4)
	service.Volumes[0] = "./channel-artifacts/genesis.block:/var/hyperledger/orderer/orderer.genesis.block"
	service.Volumes[1] = "./crypto-config/ordererOrganizations/" + domain + "/orderers/" + hostname + "/msp:/var/hyperledger/orderer/msp"
	service.Volumes[2] = "./crypto-config/ordererOrganizations/" + domain + "/orderers/" + hostname + "/tls/:/var/hyperledger/orderer/tls"
	service.Volumes[3] = "/data/orderer/" + hostname + ":/var/hyperledger/production"
	service.Privileged = true
	service.Ports = make([]string, 1)
	service.Ports[0] = "7050:7050"
	service.NetworkMode = "host"
	dc.Networks = make(map[string]*Network)
	networks := make(map[string]*Network)
	networks[net] = &Network{
		External: &External{
			Name: net,
		},
	}
	dc.Networks = networks
	dc.Services = make(map[string]*Service)
	dc.Services[serviceName] = service

	return dc
}

func genCaService(org int, domain, net, ns string) DockerCompose {
	dc := DockerCompose{
		Version: "3",
	}
	serviceName := "ca_org" + strconv.Itoa(org)
	hostname := "ca.org" + strconv.Itoa(org) + "." + domain
	service := &Service{
		Hostname: hostname,
		Dns:      make([]string, 1),
	}
	service.Networks = make(map[string]*ServNet, 1)
	service.Networks[net] = &ServNet{
		Aliases: []string{hostname},
	}
	service.Dns[0] = ns
	orgId := strconv.Itoa(org)
	service.Image = "hyperledger/fabric-ca" + TAG
	service.Environment = make([]string, 6)
	service.Environment[0] = "FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server"
	service.Environment[1] = "FABRIC_CA_SERVER_CA_NAME=" + hostname
	service.Environment[2] = "FABRIC_CA_SERVER_TLS_ENABLED=true"
	service.Environment[3] = "FABRIC_CA_SERVER_TLS_CERTFILE=/etc/hyperledger/fabric-ca-server-config/" + hostname + "-cert.pem"
	service.Environment[4] = "FABRIC_CA_SERVER_TLS_KEYFILE=/etc/hyperledger/fabric-ca-server-config/CA" + orgId + "_PRIVATE_KEY"
	service.Environment[5] = "GODEBUG=netdns=go"
	service.Command = "sh -c 'fabric-ca-server start --ca.certfile /etc/hyperledger/fabric-ca-server-config/" + hostname + "-cert.pem --ca.keyfile /etc/hyperledger/fabric-ca-server-config/CA" + orgId + "_PRIVATE_KEY -b admin:adminpw -db.type mysql -db.datasource root:rootpw@tcp\\(mysql.rds.aliyun.com:3306\\)/fabrid_ca_dbname?parseTime=true'"
	service.Volumes = make([]string, 1)
	service.Volumes[0] = "./crypto-config/peerOrganizations/org" + orgId + "." + domain + "/ca/:/etc/hyperledger/fabric-ca-server-config"
	service.Privileged = true
	service.Ports = []string{"7054:7054"}
	dc.Services = make(map[string]*Service)
	dc.Services[serviceName] = service
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
