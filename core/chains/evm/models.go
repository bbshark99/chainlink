package evm

import (
	"math/big"
	"net/url"

	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/services/eth"
	"github.com/smartcontractkit/chainlink/core/utils"
	"gopkg.in/guregu/null.v4"
)

type Chain interface {
	IsArbitrum() bool
	IsOptimism() bool
}

// TODO: Rename this to just 'Chain' and figure out what to do with the other model
type EVMChain struct {
	ID    utils.Big `gorm:"primary_key"`
	Nodes []Node
	// TODO: Add a config here which can read from database overrides but defaults to the default chain config
}

type Node struct {
	ID       int32 `gorm:"primary_key"`
	Name     string
	ChainID  utils.Big
	WSURL    string      `gorm:"column:ws_url"`
	HTTPURL  null.String `gorm:"column:http_url"`
	SendOnly bool
}

func (n Node) newPrimary() (*eth.Node, error) {
	if n.SendOnly {
		return nil, errors.New("cannot cast send-only node to primary")
	}
	wsuri, err := url.Parse(n.WSURL)
	if err != nil {
		return nil, errors.Wrap(err, "invalid websocket uri")
	}
	var httpuri *url.URL
	if n.HTTPURL.Valid {
		u, err := url.Parse(n.HTTPURL.String)
		if err != nil {
			return nil, errors.Wrap(err, "invalid http uri")
		}
		httpuri = u
	}

	return eth.NewNode(*wsuri, httpuri, n.Name), nil
}

func (n Node) newSendOnly() (*eth.SecondaryNode, error) {
	if !n.SendOnly {
		return nil, errors.New("cannot cast non send-only node to secondarynode")
	}
	if !n.HTTPURL.Valid {
		return nil, errors.New("send only node was missing HTTP url")
	}
	httpuri, err := url.Parse(n.HTTPURL.String)
	if err != nil {
		return nil, errors.Wrap(err, "invalid http uri")
	}

	return eth.NewSecondaryNode(*httpuri, n.Name), nil
}

func NewEthClientFromChain(chain EVMChain) (eth.Client, error) {
	nodes := chain.Nodes
	chainID := big.Int(chain.ID)
	var primary *eth.Node
	var sendonlys []*eth.SecondaryNode
	for _, node := range nodes {
		if node.SendOnly {
			sendonly, err := node.newSendOnly()
			if err != nil {
				return nil, err
			}
			sendonlys = append(sendonlys, sendonly)
		} else {
			var err error
			primary, err = node.newPrimary()
			if err != nil {
				return nil, err
			}
		}
	}
	return eth.NewClientWithNodes(primary, sendonlys, chainID)
}
