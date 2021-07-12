package cli

import (
	"context"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"os"
	"strings"
	"wallet/hdwallet"
	"wallet/sol"
)



type CmdClient struct {
	//区块链网络地址
	network string
	//keystore文件路径
	dataDir string
}

func NewCmdClient(network, datadir string) *CmdClient {
	return &CmdClient{
		network: network,
		dataDir: datadir,
	}
}

func (c CmdClient) Help() {
	fmt.Println("Usage:")
	fmt.Println("wallet createwallet -password PASSWORD --for create new wallet")
	fmt.Println("wallet transfer -from FROMADDR -to TOADDR -value VALUE --for transfer from acct to toaddr")
	fmt.Println("wallet getbalance -from FROMADDR --for get balance")
	fmt.Println("wallet sendtoken -from FROMADDR -to TOADDR -value VALUE --for send tokens")
	fmt.Println("wallet tokenbalance -from FROMADDR --for get tokenbalance")
	fmt.Println("wallet tokendetail -who WHO --for get tokendetail(token transfer records)")
}

//封装钱包创建方法, 该方法需要传入一个口令
func (c CmdClient) createWallet(pass string) error {
	w, err := hdwallet.NewHDWallet(c.dataDir)
	if err != nil {
		log.Panic("Failed to createWallet: ",err)
	}
	return w.StoreKey(pass)
}

//transfer方法实现交易全过程
func (c CmdClient) transfer(from, toaddr string, value int64) error {
	//1. 钱包加载
	w, _ := hdwallet.LoadWallet(from, c.dataDir)
	//2. 连接到以太坊节点
	ethcli, err := ethclient.Dial(c.network)
	if err != nil {
		log.Panic("Failed to connect Ethereum: ", err)
	}
	defer ethcli.Close()
	//3. 获取账户交易的nonce值
	nonce, _ := ethcli.NonceAt(context.Background(), common.HexToAddress(from), nil)
	//4. 创建未签名的交易
	gasLimit := uint64(300000)
	gasPrice := big.NewInt(21000000000)
	amount := big.NewInt(value)
	tx := types.NewTransaction(nonce, common.HexToAddress(toaddr), amount, gasLimit, gasPrice, []byte("Salary"))
	//5. 签名
	signedTx, err := w.HDKeystore.SignTx(common.HexToAddress(from), tx, nil)
	if err != nil {
		log.Panic("Failed to SignTx: ", err)
	}
	//6. 发送交易
	return ethcli.SendTransaction(context.Background(), signedTx)
}

func (c CmdClient) getBalance(from string) (int64, error) {
	//1. 连接至以太坊
	ethcli, err := ethclient.Dial(c.network)
	if err != nil {
		log.Panic("Failed to connect Ethereum: ", err)
	}
	defer ethcli.Close()
	//2. 查询余额
	addr := common.HexToAddress(from)
	value, err := ethcli.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		log.Panic("Failed to BalanceAt: ", err)
	}
	return value.Int64(), nil
}

const LelecoinContractAddr = "0x9B4E5A473d60D2D696F82d224723769d25F104c2"
func (c CmdClient) sendToken(from, toaddr string, value int64) error {
	//1. 连接以太坊
	cli, _ := ethclient.Dial(c.network)
	fmt.Println("connect success")
	defer cli.Close()
	//2. 创建Token合约实例, 需要合约地址
	lelecoin, _ := sol.NewLelecoin(common.HexToAddress(LelecoinContractAddr), cli)
	//fmt.Println("newlelecoin success")
	//3. 设置调用身份
	//3.1 钱包加载
	w, _ := hdwallet.LoadWallet(from, c.dataDir)
	//3.2 利用钱包私钥创建身份
	nonce, err := cli.PendingNonceAt(context.Background(), common.HexToAddress(from))
	if err != nil {
		log.Fatal(err)
	}
	gasPrice, err := cli.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	auth := w.HDKeystore.NewTransactOpts()
	auth.Nonce = big.NewInt(int64(nonce))
	//注意必须设定auth.Context, 否则报错-> nil Context!!!
	auth.Context = context.Background()

	auth.Value = big.NewInt(0)     // in wei
	auth.GasLimit = uint64(300000) // in units
	//注意必须设置auth.GasPrice, 否则也会报错!!!
	auth.GasPrice = gasPrice

	//fmt.Println("newtxopts success")
	//4. 调用transfer
	_, err = lelecoin.Transfer(auth, common.HexToAddress(toaddr), big.NewInt(value))
	return err
}

func (c CmdClient) tokenbalance(from string) (int64, error) {
	//1. 连接以太坊
	cli, err := ethclient.Dial(c.network)
	if err != nil {
		log.Panic("Failed to Dial: ", err)
	}
	defer cli.Close()
	//2. 创建Token合约实例, 需要合约地址
	lelecoin, err := sol.NewLelecoin(common.HexToAddress(LelecoinContractAddr), cli)
	if err != nil {
		log.Panic("Failed to NewLelecoin: ", err)
	}
	//3. 构建CallOpts
	fromAddr := common.HexToAddress(from)
	opts := bind.CallOpts{
		Pending:     false,
		From:        fromAddr,
		BlockNumber: nil,
		Context:     nil,
	}

	value, err := lelecoin.BalanceOf(&opts, fromAddr)
	if err != nil {
		log.Panic("Failed to call BalanceOf: ", err)
	}
	fmt.Printf("%s's token balance is: %d\n", from, value.Int64())
	return value.Int64(), err
}

func (c CmdClient) tokendetail(who string) error {
	//1. connect blockchain network
	cli, err := ethclient.Dial(c.network)
	if err != nil {
		log.Panic("Failed to Dial: ", err)
	}
	defer cli.Close()
	//2. 设置过滤条件
	contractAddr := common.HexToAddress(LelecoinContractAddr)
	topicHash := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

	query := ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: nil,
		ToBlock:   nil,
		Addresses: []common.Address{contractAddr},
		Topics:    [][]common.Hash{{topicHash}},
	}

	logs, err := cli.FilterLogs(context.Background(), query)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Transfer records of address %s\n", who)
	for _, vLog := range logs {
		//再验证一次vLog有效性
		if contractAddr == vLog.Address && len(vLog.Topics) == 3 && vLog.Topics[0] == topicHash {
			from := vLog.Topics[1].Bytes()[len(vLog.Topics[1].Bytes()) - 20:]
			to := vLog.Topics[2].Bytes()[len(vLog.Topics[2].Bytes()) - 20:]
			val := big.NewInt(0)
			val.SetBytes(vLog.Data)
			//注意：以太坊地址在网络中是忽略大小写的
			//但是golang在字符串比较时需要精准匹配大小写
			//这里使用strings.ToUpper统一转为大写进行比较
			if strings.ToUpper(fmt.Sprintf("0x%x", from)) == strings.ToUpper(who) {
				fmt.Printf("\tfrom: 0x%x\n\tto: 0x%x\n\tvalue: -%d lelecoin\n\tBlockNumber: %d\n", from, to, val.Int64(), vLog.BlockNumber)
				fmt.Println()
			}
			if strings.ToUpper(fmt.Sprintf("0x%x", to)) == strings.ToUpper(who) {
				fmt.Printf("\tfrom: 0x%x\n\tto: 0x%x\n\tvalue: +%d lelecoin\n\tBlockNumber: %d\n", from, to, val.Int64(), vLog.BlockNumber)
				fmt.Println()
			}
		}
	}
	return nil
}

func (c CmdClient) Run() {
	//判断参数是否准确
	if len(os.Args) < 2 {
		fmt.Println("Unknown Command")
		fmt.Println("Run 'wallet help' for usage")
		os.Exit(-1)
	}

	//1. 立flag
	helpCmd := flag.NewFlagSet("help", flag.ExitOnError)
	cwCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	transferCmd := flag.NewFlagSet("transfer", flag.ExitOnError)
	getbalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	sendtokenCmd := flag.NewFlagSet("sendtoken", flag.ExitOnError)
	tokenbalanceCmd := flag.NewFlagSet("tokenbalance", flag.ExitOnError)
	tokendetailCmd := flag.NewFlagSet("tokendetail", flag.ExitOnError)

	//2. 立flag参数
	cwCmdPw := cwCmd.String("password", "", "PASSWORD")

	transferCmdFrom := transferCmd.String("from", "", "FROMADDR")
	transferCmdTo := transferCmd.String("to", "", "TOADDR")
	transferCmdValue := transferCmd.Int64("value", 0, "VALUE")

	getbalanceCmdFrom := getbalanceCmd.String("from", "", "FROMADDR")

	sendtokenCmdFrom := sendtokenCmd.String("from", "", "FROMADDR")
	sendtokenCmdTo := sendtokenCmd.String("to", "", "TOADDR")
	sendtokenCmdValue := sendtokenCmd.Int64("value", 0, "VALUE")

	tokenbalanceCmdFrom := tokenbalanceCmd.String("from", "", "FROMADDR")

	detailCmdWho := tokendetailCmd.String("who", "", "WHO")
	//3. 解析命令行参数
	switch os.Args[1] {
	case "help":
		err := helpCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to Parse help flag: ", err)
			return
		}
	case "createwallet":
		err := cwCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to Parse createwallet flag: ", err)
			return
		}
	case "transfer":
		err := transferCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to Parse transfer flag: ", err)
			return
		}
	case "getbalance":
		err := getbalanceCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to Parse getbalance flag: ", err)
			return
		}
	case "sendtoken":
		err := sendtokenCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to Parse sendtoken flag: ", err)
			return
		}
	case "tokenbalance":
		err := tokenbalanceCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to Parse tokenbalance: ", err)
			return
		}
	case "tokendetail":
		err := tokendetailCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println("Failed to Parse tokendetail: ", err)
		}
	default:
		fmt.Println("Unknown Command")
		fmt.Println("Run 'wallet help' for usage")
	}

	//4. 确认flag参数出现
	if helpCmd.Parsed() {
		c.Help()
	}

	if cwCmd.Parsed() {
		fmt.Printf("password: %s\n", *cwCmdPw)
		err := c.createWallet(*cwCmdPw)
		if err != nil {
			log.Panic("Failed to createWallet: ", err)
		}
	}

	if transferCmd.Parsed() {
		fmt.Printf("from: %s, to: %s, value: %d\n", *transferCmdFrom, *transferCmdTo, *transferCmdValue)
		err := c.transfer(*transferCmdFrom, *transferCmdTo, *transferCmdValue)
		if err != nil {
			log.Panic("Failed to transfer: ", err)
		}
		fmt.Println("Success")
	}

	if getbalanceCmd.Parsed() {
		fmt.Printf("from: %s\n", *getbalanceCmdFrom)
		value, err := c.getBalance(*getbalanceCmdFrom)
		if err != nil {
			log.Panic("Failed to getBalance: ", err)
		}
		fmt.Printf("%s's balance is %d\n", *getbalanceCmdFrom, value)
	}

	if sendtokenCmd.Parsed() {
		fmt.Printf("from: %s, to: %s, value: %d\n", *sendtokenCmdFrom, *sendtokenCmdTo, *sendtokenCmdValue)
		err := c.sendToken(*sendtokenCmdFrom, *sendtokenCmdTo, *sendtokenCmdValue)
		if err != nil {
			log.Panic("Failed to sendtoken: ", err)
		}
		fmt.Println("Success")
	}

	if tokenbalanceCmd.Parsed() {
		_, _ = c.tokenbalance(*tokenbalanceCmdFrom)
	}

	if tokendetailCmd.Parsed() {
		_ = c.tokendetail(*detailCmdWho)
	}
}

