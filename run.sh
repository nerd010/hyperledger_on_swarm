#!/bin/bash
set -e 

export PATH=${PWD}/bin:${PWD}:$PATH
export FABRIC_CFG_PATH=${PWD}
export VERSION="1.0.2"
export ARCH=$(echo "$(uname -s|tr '[:upper:]' '[:lower:]'|sed 's/mingw64_nt.*/windows/')-$(uname -m | sed 's/x86_64/amd64/g')" | awk '{print tolower($0)}')
#Set MARCH variable i.e ppc64le,s390x,x86_64,i386
MARCH=`uname -m`

CHANNEL="chaichannel"
KAFKA=4
ZOOKEEPER=3
ORDERER=3
PEER=2
ORGS=2
DOMAIN="chai.cn"
SWARM=0
BINARY=0


function printHelp(){
  echo "Usage: "
  echo "  run.sh [-k <kafka num> ] [-z <zookeeper number>] [-o <orderer number>] [-p <prodution 0 or 1>] [-g <org number> ]  [-d <domain name>] [ -s <swarm mode  0 or 1>] [-v <fabric version> ] [ -b <download binary 0 or 1> ] [ -c <channel name> ]"
  echo "  run.sh -h|--help (print this message)"
  echo "    -o <orderer number> - orderer number (default to 3)"
  echo "    -k <kafka number> kafka number (default to 4)"
  echo "    -g <org number> org number (default to 2)"
  echo "    -c <channel name> - channel name to use (defaults to \"mychannel\")"
  echo "    -d <domain> - domain name (defaults to chai.cn)"
  echo "    -s <swarm mode> - for swarm mode set 1 (defaults to 0)"
  echo "    -b <download binary> - download cryptogen and configtxgen tools (defaults to 0)"
  echo "    -v <fabric image version> "
}

function downloadBinary(){
    echo "===> Downloading platform binaries"
    curl https://nexus.hyperledger.org/content/repositories/releases/org/hyperledger/fabric/hyperledger-fabric/${ARCH}-${VERSION}/hyperledger-fabric-${ARCH}-${VERSION}.tar.gz | tar xz
}

function generateComposefiles(){
    if [ "$SWARM" == "0" ];then
        ./bin/$MARCH/genConfig -Kafka $KAFKA -Orderer $ORDERER -Orgs $ORGS -Peer $PEER  -Zookeeper $ZOOKEEPER -domain $DOMAIN -tagtail $VERSION
    else
         ./bin/$MARCH/genConfig -Kafka $KAFKA -Orderer $ORDERER -Orgs $ORGS -Peer $PEER  -Zookeeper $ZOOKEEPER -domain $DOMAIN -tagtail $VERSION -dev -prod
    fi;
}

function replacePrivateKey() {
    ARCH=`uname -s | grep Darwin`
  if [ "$ARCH" == "Darwin" ]; then
    OPTS="-it"
  else
    OPTS="-i"
  fi

  # Copy the template to the file that will be modified to add the private key
  

  # The next steps will replace the template's contents with the
  # actual values of the private key file names for the two CAs.
  CURRENT_DIR=$PWD
  for i in `seq 1 $ORGS`
  do
    cd crypto-config/peerOrganizations/org${i}.${DOMAIN}/ca/
    PRIV_KEY=$(ls *_sk)
    cd $CURRENT_DIR
    if [ "$SWARM" == "0" ];then
        sed $OPTS "s/CA${i}_PRIVATE_KEY/${PRIV_KEY}/g" ca.org${i}.${DOMAIN}.yaml
    else
        sed $OPTS "s/CA${i}_PRIVATE_KEY/${PRIV_KEY}/g" hyperledger-ca.yaml 
    fi;
    if [ "$ARCH" == "Darwin" ]; then
        rm  -rf ca.org${i}.${DOMAIN}.yamlt;
        rm  -rf hyperledger-ca.yamlt;
    fi;
  done
}

function generateCerts()
{
  rm -rf crypto-config/*
  which cryptogen
  if [ "$?" -ne 0 ]; then
    echo "cryptogen tool not found. exiting"
    exit 1
  fi
  echo
  echo "##########################################################"
  echo "##### Generate certificates using cryptogen tool #########"
  echo "##########################################################"

  cryptogen generate --config=./crypto-config.yaml
  if [ "$?" -ne 0 ]; then
    echo "Failed to generate certificates..."
    exit 1
  fi
  echo
    
}


function generateChannelArtifacts(){
    rm -rf channel-artifacts/*
    which configtxgen
  if [ "$?" -ne 0 ]; then
    echo "configtxgen tool not found. exiting"
    exit 1
  fi

  echo "##########################################################"
  echo "#########  Generating Orderer Genesis block ##############"
  echo "##########################################################"
  # Note: For some unknown reason (at least for now) the block file can't be
  # named orderer.genesis.block or the orderer will fail to launch!
  configtxgen -profile TwoOrgsOrdererGenesis -outputBlock ./channel-artifacts/genesis.block
  if [ "$?" -ne 0 ]; then
    echo "Failed to generate orderer genesis block..."
    exit 1
  fi
  echo
  echo "#################################################################"
  echo "### Generating channel configuration transaction 'channel.tx' ###"
  echo "#################################################################"
  configtxgen -profile TwoOrgsChannel -outputCreateChannelTx ./channel-artifacts/channel.tx -channelID $CHANNEL_NAME
  if [ "$?" -ne 0 ]; then
    echo "Failed to generate channel configuration transaction..."
    exit 1
  fi

    i=1
    while [ "$i" -le "ORGS" ]; do
        echo
        echo "#################################################################"
        echo "#######    Generating anchor peer update for Org${i}MSP   ##########"
        echo "#################################################################"
        $CONFIGTXGEN -profile TwoOrgsChannel -outputAnchorPeersUpdate ./channel-artifacts/Org${i}MSPanchors.tx -channelID $CHANNEL_NAME -asOrg Org${i}MSP
        if [ "$?" -ne 0 ]; then
            echo "Failed to generate anchor peer update for Org${i}MSP..."
            exit 1
        fi
        i=$(($i + 1))
    done
  
    
}


function generateScripts(){
    mkdir -p scripts 
    cat > scripts/switchenv.sh <<EOF
function switchEnv()
{
    if [ $# != 3 ];then
        echo "Usage: switchEnv PeerNO OrgNO Domain"
        exit 1;
    fi;
    PEER=$1
    ORG=$2
    DOMAIN=$3
    CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org${ORG}.${DOMAIN}/users/Admin@org${ORG}.${DOMAIN}/msp
    CORE_PEER_ADDRESS=peer${PEER}.org${ORG}.${DOMAIN}:7051
    CORE_PEER_LOCALMSPID="Org${ORG}MSP"
    CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org${ORG}.${DOMAIN}/peers/peer${PEER}.org${ORG}.${DOMAIN}/tls/ca.crt
}
EOF
}

while getopts "h?k:z:o:p:g:d:s:v:b:c" 
do
    case "$opt" in 
        h|\?)
            printHelp
            exit 0
        ;;
        k) KAFKA=$OPTARG
        ;;
        z) ZOOKEEPER=$OPTARG
        ;;
        o) ORDERER=$OPTARG
        ;;
        p) PEER=$OPTARG
        ;;
        g) ORGS=$OPTARG
        ;;
        d) DOMAIN=$OPTARG
        ;;
        s) SWARM=$OPTARG
        ;;
        v) VERSION=$OPTARG
        ;;
        b) BINARY=$OPTARG
            if [ "$BINARY" == "1" ];then 
                downloadBinary
                exit 0
            fi;
        ;;
        c) CHANNEL=$OPTARG
        ;;
    esac
done

generateComposefiles
replacePrivateKey

echo "===> Downloading platform binaries"
curl https://nexus.hyperledger.org/content/repositories/releases/org/hyperledger/fabric/hyperledger-fabric/${ARCH}-${VERSION}/hyperledger-fabric-${ARCH}-${VERSION}.tar.gz | tar xz
sh genConfig/build.sh

mkdir -p scripts

./bin/$ARCH/genCOnfig -Kafka 4 -Orderer 3 -Orgs 2 -Peer 2 -Zookeeper 3 -domain chai.cn -dev -prod -tagtail $VERSION
