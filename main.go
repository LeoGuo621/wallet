package main

import "wallet/cli"

func main() {
	//w, err := hdwallet.NewHDWallet("./keystore")
	//if err != nil {
	//	fmt.Println("Failed to NewWallet: ", err)
	//	return
	//}
	//w.StoreKey("123")

	c := cli.NewCmdClient("http://localhost:8545", "./keystore")
	c.Run()
}