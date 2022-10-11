package upgrader

import (
	"fmt"

	"github.com/dogechain-lab/dogechain/chain"
	"github.com/dogechain-lab/dogechain/contracts/systemcontracts"
	"github.com/dogechain-lab/dogechain/helper/hex"
	"github.com/dogechain-lab/dogechain/state"
	"github.com/dogechain-lab/dogechain/types"
	"github.com/hashicorp/go-hclog"
)

type UpgradeConfig struct {
	ContractAddr types.Address
	CommitURL    string
	Code         string
}

type Upgrade struct {
	UpgradeName string
	Configs     []*UpgradeConfig
}

// type upgradeHook func(blockNumber uint64, contractAddr types.Address, statedb *state.State) error

const (
	MainNetChainID = 2000
)

const (
	mainNet = "Mainnet"
)

var (
	GenesisHash types.Hash
	//upgrade config
	// portland
	_portlandUpgrade = make(map[string]*Upgrade)
	// detroit
	_detroitUpgrade = make(map[string]*Upgrade)
)

func init() {
	//nolint:lll
	_portlandUpgrade[mainNet] = &Upgrade{
		UpgradeName: "portland",
		Configs: []*UpgradeConfig{
			{
				ContractAddr: systemcontracts.AddrBridgeContract,
				CommitURL:    "https://github.com/dogechain-lab/contracts/commit/bcaad0a8a050743855d294d58dac73f06fdc9585",
				Code:         "6080604052600436106101085760003560e01c8063715018a6116100955780639dc29fac116100645780639dc29fac14610321578063cd86a6cb1461034a578063d91921ed14610387578063eb12d61e146103b0578063f2fde38b146103d957610108565b8063715018a6146102775780637df73e271461028e5780638da5cb5b146102cb57806394cf795e146102f657610108565b806331fb67c2116100dc57806331fb67c2146101b557806334fcf437146101d15780634cde3a53146101fa57806354c4633e1461022557806367058d291461024e57610108565b8062a8efc71461010d57806318160ddd1461013657806319e5c034146101615780632c4e722e1461018a575b600080fd5b34801561011957600080fd5b50610134600480360381019061012f919061198e565b610402565b005b34801561014257600080fd5b5061014b6104ae565b6040516101589190611d88565b60405180910390f35b34801561016d57600080fd5b50610188600480360381019061018391906118a6565b6104b8565b005b34801561019657600080fd5b5061019f61082a565b6040516101ac9190611d88565b60405180910390f35b6101cf60048036038101906101ca9190611945565b610834565b005b3480156101dd57600080fd5b506101f860048036038101906101f3919061198e565b61092d565b005b34801561020657600080fd5b5061020f610a12565b60405161021c9190611d88565b60405180910390f35b34801561023157600080fd5b5061024c60048036038101906102479190611839565b610a1c565b005b34801561025a57600080fd5b506102756004803603810190610270919061198e565b610b63565b005b34801561028357600080fd5b5061028c610c48565b005b34801561029a57600080fd5b506102b560048036038101906102b09190611839565b610ce2565b6040516102c29190611c94565b60405180910390f35b3480156102d757600080fd5b506102e0610d38565b6040516102ed9190611c57565b60405180910390f35b34801561030257600080fd5b5061030b610d61565b6040516103189190611c72565b60405180910390f35b34801561032d57600080fd5b5061034860048036038101906103439190611866565b610def565b005b34801561035657600080fd5b50610371600480360381019061036c91906118a6565b610ee0565b60405161037e9190611c72565b60405180910390f35b34801561039357600080fd5b506103ae60048036038101906103a9919061198e565b610fb9565b005b3480156103bc57600080fd5b506103d760048036038101906103d29190611839565b611065565b005b3480156103e557600080fd5b5061040060048036038101906103fb9190611839565b6112a6565b005b3373ffffffffffffffffffffffffffffffffffffffff1660008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614610490576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161048790611d28565b60405180910390fd5b6104a5816006546113b090919063ffffffff16565b60068190555050565b6000600654905090565b600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff16610544576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161053b90611d68565b60405180910390fd5b60008484848460405160200161055d9493929190611c11565b60405160208183030381529060405280519060200120905060006005600083815260200190815260200160002090508060040160149054906101000a900460ff16156105aa575050610824565b60005b816000018054905081101561064b573373ffffffffffffffffffffffffffffffffffffffff168260000182815481106105e9576105e86121a3565b5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141561063857505050610824565b808061064390612070565b9150506105ad565b50858160040160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550848160010181905550828160030190805190602001906106b09291906116fc565b50838160020190805190602001906106c99291906116fc565b5080600001339080600181540180825580915050600190039060005260206000200160009091909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550600280805490506107409190611ec4565b816000018054905011801561076457508060040160149054906101000a900460ff16155b156108215760018160040160146101000a81548160ff02191690831515021790555061079b856006546113c690919063ffffffff16565b60068190555080600101548160040160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167fbceab28ca952a9177ce3716580d6c8c2d677fdf721b944e57a5e7322622ffdc98360020184600301604051610818929190611cd1565b60405180910390a35b50505b50505050565b6000600754905090565b600154341015610879576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161087090611d48565b60405180910390fd5b60006108a4612710610896600754346113dc90919063ffffffff16565b6113f290919063ffffffff16565b905060006108bb82346113b090919063ffffffff16565b90506108d2816006546113b090919063ffffffff16565b60068190555081813373ffffffffffffffffffffffffffffffffffffffff167f62116a798bb58cc967874bea4d771de2f9aeec6c64189ff2e5a551072f3106f9866040516109209190611caf565b60405180910390a4505050565b3373ffffffffffffffffffffffffffffffffffffffff1660008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16146109bb576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016109b290611d28565b60405180910390fd5b600060075490508160078190555081813373ffffffffffffffffffffffffffffffffffffffff167f9e31cca092b9e764bfc6b1b552d55ad4b035e609318fecc26cd38b34e8dd08bb60405160405180910390a45050565b6000600154905090565b3373ffffffffffffffffffffffffffffffffffffffff1660008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614610aaa576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610aa190611d28565b60405180910390fd5b600360008273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff1615610b6057610b0581611408565b8073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f013d6b862b532c38b01efed34c94d382085143963c63c76e87c24d4b7a37f98e60405160405180910390a35b50565b3373ffffffffffffffffffffffffffffffffffffffff1660008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614610bf1576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610be890611d28565b60405180910390fd5b600060015490508160018190555081813373ffffffffffffffffffffffffffffffffffffffff167f480e8e496f7aff74972b0902e678fd5b564e4fb6527f0418da8a2c1aa628002260405160405180910390a45050565b3373ffffffffffffffffffffffffffffffffffffffff1660008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614610cd6576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610ccd90611d28565b60405180910390fd5b610ce06000611638565b565b6000600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff169050919050565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60606002805480602002602001604051908101604052809291908181526020018280548015610de557602002820191906000526020600020905b8160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019060010190808311610d9b575b5050505050905090565b3373ffffffffffffffffffffffffffffffffffffffff1660008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614610e7d576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610e7490611d28565b60405180910390fd5b610e92816006546113b090919063ffffffff16565b600681905550808273ffffffffffffffffffffffffffffffffffffffff167f696de425f79f4a40bc6d2122ca50507f0efbeabbff86a84871b7196ab8ea8df760405160405180910390a35050565b6060600085858585604051602001610efb9493929190611c11565b60405160208183030381529060405280519060200120905060056000828152602001908152602001600020600001805480602002602001604051908101604052809291908181526020018280548015610fa957602002820191906000526020600020905b8160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019060010190808311610f5f575b5050505050915050949350505050565b3373ffffffffffffffffffffffffffffffffffffffff1660008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614611047576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161103e90611d28565b60405180910390fd5b61105c816006546113c690919063ffffffff16565b60068190555050565b3373ffffffffffffffffffffffffffffffffffffffff1660008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16146110f3576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016110ea90611d28565b60405180910390fd5b600360008273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff166112a357600280549050600460008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506001600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff0219169083151502179055506002819080600181540180825580915050600190039060005260206000200160009091909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8064a302796c89446a96d63470b5b036212da26bd2debe5bec73e0170a9a5e8360405160405180910390a35b50565b3373ffffffffffffffffffffffffffffffffffffffff1660008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614611334576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161132b90611d28565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614156113a4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161139b90611d08565b60405180910390fd5b6113ad81611638565b50565b600081836113be9190611f4f565b905092915050565b600081836113d49190611e6e565b905092915050565b600081836113ea9190611ef5565b905092915050565b600081836114009190611ec4565b905092915050565b6000600460008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050600060016002805490506114609190611f4f565b905080821461154f5760006002828154811061147f5761147e6121a3565b5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905080600284815481106114c1576114c06121a3565b5b9060005260206000200160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555082600460008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550505b6000600360008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff0219169083151502179055506000600460008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555060028054806115fe576115fd612174565b5b6001900381819060005260206000200160006101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690559055505050565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050816000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b8280546117089061200d565b90600052602060002090601f01602090048101928261172a5760008555611771565b82601f1061174357805160ff1916838001178555611771565b82800160010185558215611771579182015b82811115611770578251825591602001919060010190611755565b5b50905061177e9190611782565b5090565b5b8082111561179b576000816000905550600101611783565b5090565b60006117b26117ad84611dc8565b611da3565b9050828152602081018484840111156117ce576117cd612206565b5b6117d9848285611fcb565b509392505050565b6000813590506117f0816122fd565b92915050565b600082601f83011261180b5761180a612201565b5b813561181b84826020860161179f565b91505092915050565b60008135905061183381612314565b92915050565b60006020828403121561184f5761184e612210565b5b600061185d848285016117e1565b91505092915050565b6000806040838503121561187d5761187c612210565b5b600061188b858286016117e1565b925050602061189c85828601611824565b9150509250929050565b600080600080608085870312156118c0576118bf612210565b5b60006118ce878288016117e1565b94505060206118df87828801611824565b935050604085013567ffffffffffffffff811115611900576118ff61220b565b5b61190c878288016117f6565b925050606085013567ffffffffffffffff81111561192d5761192c61220b565b5b611939878288016117f6565b91505092959194509250565b60006020828403121561195b5761195a612210565b5b600082013567ffffffffffffffff8111156119795761197861220b565b5b611985848285016117f6565b91505092915050565b6000602082840312156119a4576119a3612210565b5b60006119b284828501611824565b91505092915050565b60006119c783836119d3565b60208301905092915050565b6119dc81611f83565b82525050565b6119eb81611f83565b82525050565b611a026119fd82611f83565b6120b9565b82525050565b6000611a1382611e1e565b611a1d8185611e41565b9350611a2883611df9565b8060005b83811015611a59578151611a4088826119bb565b9750611a4b83611e34565b925050600181019050611a2c565b5085935050505092915050565b611a6f81611f95565b82525050565b6000611a8082611e29565b611a8a8185611e52565b9350611a9a818560208601611fda565b611aa381612215565b840191505092915050565b6000611ab982611e29565b611ac38185611e63565b9350611ad3818560208601611fda565b80840191505092915050565b60008154611aec8161200d565b611af68186611e52565b94506001821660008114611b115760018114611b2357611b56565b60ff1983168652602086019350611b56565b611b2c85611e09565b60005b83811015611b4e57815481890152600182019150602081019050611b2f565b808801955050505b50505092915050565b6000611b6c602683611e52565b9150611b7782612233565b604082019050919050565b6000611b8f601c83611e52565b9150611b9a82612282565b602082019050919050565b6000611bb2600683611e52565b9150611bbd826122ab565b602082019050919050565b6000611bd5601d83611e52565b9150611be0826122d4565b602082019050919050565b611bf481611fc1565b82525050565b611c0b611c0682611fc1565b6120dd565b82525050565b6000611c1d82876119f1565b601482019150611c2d8286611bfa565b602082019150611c3d8285611aae565b9150611c498284611aae565b915081905095945050505050565b6000602082019050611c6c60008301846119e2565b92915050565b60006020820190508181036000830152611c8c8184611a08565b905092915050565b6000602082019050611ca96000830184611a66565b92915050565b60006020820190508181036000830152611cc98184611a75565b905092915050565b60006040820190508181036000830152611ceb8185611adf565b90508181036020830152611cff8184611adf565b90509392505050565b60006020820190508181036000830152611d2181611b5f565b9050919050565b60006020820190508181036000830152611d4181611b82565b9050919050565b60006020820190508181036000830152611d6181611ba5565b9050919050565b60006020820190508181036000830152611d8181611bc8565b9050919050565b6000602082019050611d9d6000830184611beb565b92915050565b6000611dad611dbe565b9050611db9828261203f565b919050565b6000604051905090565b600067ffffffffffffffff821115611de357611de26121d2565b5b611dec82612215565b9050602081019050919050565b6000819050602082019050919050565b60008190508160005260206000209050919050565b600081519050919050565b600081519050919050565b6000602082019050919050565b600082825260208201905092915050565b600082825260208201905092915050565b600081905092915050565b6000611e7982611fc1565b9150611e8483611fc1565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff03821115611eb957611eb86120e7565b5b828201905092915050565b6000611ecf82611fc1565b9150611eda83611fc1565b925082611eea57611ee9612116565b5b828204905092915050565b6000611f0082611fc1565b9150611f0b83611fc1565b9250817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0483118215151615611f4457611f436120e7565b5b828202905092915050565b6000611f5a82611fc1565b9150611f6583611fc1565b925082821015611f7857611f776120e7565b5b828203905092915050565b6000611f8e82611fa1565b9050919050565b60008115159050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b82818337600083830152505050565b60005b83811015611ff8578082015181840152602081019050611fdd565b83811115612007576000848401525b50505050565b6000600282049050600182168061202557607f821691505b6020821081141561203957612038612145565b5b50919050565b61204882612215565b810181811067ffffffffffffffff82111715612067576120666121d2565b5b80604052505050565b600061207b82611fc1565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8214156120ae576120ad6120e7565b5b600182019050919050565b60006120c4826120cb565b9050919050565b60006120d682612226565b9050919050565b6000819050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600080fd5b600080fd5b600080fd5b600080fd5b6000601f19601f8301169050919050565b60008160601b9050919050565b7f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160008201527f6464726573730000000000000000000000000000000000000000000000000000602082015250565b7f4f6e6c79206f776e65722063616e2063616c6c2066756e6374696f6e00000000600082015250565b7f466f726269640000000000000000000000000000000000000000000000000000600082015250565b7f4f6e6c79207369676e65722063616e2063616c6c2066756e6374696f6e000000600082015250565b61230681611f83565b811461231157600080fd5b50565b61231d81611fc1565b811461232857600080fd5b5056fea26469706673582212208879968f26c5ad21d7e3b58081ca73de324deb7256e1760798788f290bffec8364736f6c63430008060033",
			},
		},
	}
	//nolint:lll
	_detroitUpgrade[mainNet] = &Upgrade{
		UpgradeName: "detroit",
		Configs: []*UpgradeConfig{
			{
				ContractAddr: systemcontracts.AddrValidatorSetContract,
				CommitURL:    "https://github.com/dogechain-lab/contracts/commit/bcaad0a8a050743855d294d58dac73f06fdc9585",
				Code:         "608060405234801561001057600080fd5b50600436106102a05760003560e01c80638da5cb5b11610167578063c96be4cb116100ce578063f340fa0111610087578063f340fa0114610615578063f90ecacc14610628578063facd743b146103e3578063fd68f6f01461063b578063fd9275db1461064e578063fda259e01461066157600080fd5b8063c96be4cb146105ac578063ca1e7819146105bf578063ced5bcc1146105d4578063d3553f37146105dc578063e1a2e863146105ef578063f2fde38b1461060257600080fd5b8063aea0e78b11610120578063aea0e78b14610550578063b46e552014610558578063bb872b4a1461056b578063be1997381461057e578063c2a45d4d14610586578063c2a672e01461059957600080fd5b80638da5cb5b146104f65780638f73a58b146105075780639abee7d01461050f5780639dbf97db14610522578063ab033ea91461052a578063adc9772e1461053d57600080fd5b806342ad55ac1161020b5780636f856847116101c45780636f856847146104ae57806373a3dda6146104b6578063750142e6146104c9578063751bf202146104d257806376671808146104e55780638ae39cac146104ed57600080fd5b806342ad55ac146103e35780634878f40114610406578063500a1564146104195780635aa6e6751461043e5780635e80536a146104515780636cbe6cd8146104a657600080fd5b8063201467741161025d578063201467741461039257806325646e1f146103a557806332cc6f08146103b8578063346c90a8146103c0578063373d6132146103c857806340a141ff146103d057600080fd5b80630397d458146102a5578063097475f7146102ba5780630db14e95146103385780631d48b76a146103595780631e83409a1461036c5780631fe976841461037f575b600080fd5b6102b86102b3366004612194565b610674565b005b61031b6102c8366004612194565b60186020526000908152604090208054600182015460028301546003840154600485015460058601546006909601546001600160a01b039095169593949293919290919060ff8082169161010090041688565b60405161032f9897969594939291906122b9565b60405180910390f35b61034b6103463660046121af565b6106f9565b60405190815260200161032f565b6102b8610367366004612261565b610726565b6102b861037a366004612194565b610789565b6102b861038d366004612194565b61094d565b6102b86103a0366004612261565b610a44565b6102b86103b3366004612293565b610aa7565b600e5461034b565b600f5461034b565b60085461034b565b6102b86103de366004612194565b610b12565b6103f66103f1366004612194565b610d06565b604051901515815260200161032f565b6102b8610414366004612261565b610d13565b6009546001600160a01b03165b6040516001600160a01b03909116815260200161032f565b600a54610426906001600160a01b031681565b61048b61045f3660046121af565b601960209081526000928352604080842090915290825290208054600182015460029092015490919083565b6040805193845260208401929092529082015260600161032f565b60125461034b565b60145461034b565b6102b86104c4366004612194565b610d76565b61034b60175481565b6102b86104e0366004612261565b610eab565b61034b610f0e565b61034b60165481565b6001546001600160a01b0316610426565b60155461034b565b6102b861051d3660046121e2565b610f28565b60105461034b565b6102b8610538366004612194565b61111a565b6102b861054b3660046121e2565b611196565b61034b6113c4565b6102b8610566366004612194565b6113d9565b6102b8610579366004612261565b611475565b60115461034b565b6102b8610594366004612261565b6114d8565b6102b86105a73660046121e2565b61153b565b6102b86105ba366004612194565b611722565b6105c76118d7565b60405161032f9190612326565b60135461034b565b6105c76105ea36600461220c565b6118e3565b6102b86105fd366004612261565b6119d1565b6102b8610610366004612194565b611a34565b6102b8610623366004612194565b611acf565b610426610636366004612261565b611c1c565b61034b610649366004612194565b611c46565b6102b861065c366004612261565b611c67565b6102b861066f366004612194565b611cca565b6001546001600160a01b031633146106a75760405162461bcd60e51b815260040161069e90612373565b60405180910390fd5b600980546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f3792de2485bcc1adb436c37819f81339378f5ec98f5b3534f7b71b70b5d31c0390600090a35050565b6001600160a01b038083166000908152601960209081526040808320938516835292905220545b92915050565b6001546001600160a01b031633146107505760405162461bcd60e51b815260040161069e90612373565b6015805490829055604051829082907f0feb8e082808c8bbfe2c4004506e31819f9c22f316e82f6889e93187dd74fc7a90600090a35050565b6001600160a01b0381166000908152601960209081526040808320338452909152902080546107ed5760405162461bcd60e51b815260206004820152601060248201526f6e6f7468696e6720746f20636c61696d60801b604482015260640161069e565b6001600160a01b03808316600090815260186020908152604080832081516101008101835281549095168552600181015492850192909252600282015490840152600380820154606085015260048201546080850152600582015460a0850152600682015492939260c084019160ff9091169081111561086f5761086f61246f565b60038111156108805761088061246f565b815260069190910154610100900460ff1615156020909101526001830154608082015184549293506000926108d192916108cb9164e8d4a51000916108c59190611d44565b90611d57565b90611d63565b90506108f764e8d4a510006108c584608001518660000154611d4490919063ffffffff16565b60018401556109063382611d6f565b6040518181526001600160a01b0385169033907f70eb43c4a8ae8c40502dcf22436c509c28d6ff421cf07c491be56984bd987068906020015b60405180910390a350505050565b6001546001600160a01b031633146109775760405162461bcd60e51b815260040161069e90612373565b6001600160a01b03811660009081526018602052604090206002600682015460ff1660038111156109aa576109aa61246f565b146109c75760405162461bcd60e51b815260040161069e906123aa565b6109d2600b83611ee3565b506006810180546001919060ff191682805b021790555060068101546001600160a01b038316907f0c5c3450e67dce49a08116ce351f3a67c5b7d3217c607162d91189a7229174ea9060ff166003811115610a2f57610a2f61246f565b60405190815260200160405180910390a25050565b6001546001600160a01b03163314610a6e5760405162461bcd60e51b815260040161069e90612373565b600e805490829055604051829082907f3d657b82c31a672b7a8765b72f6f5e966cfb980ed039570a39ccdd70bf19c26690600090a35050565b6001546001600160a01b03163314610ad15760405162461bcd60e51b815260040161069e90612373565b6013805463ffffffff83169182905560405190919082907f7414a33d82c698855ef5ed249e10e2f7481971f83f98cee8d7023f15ae0e881f90600090a35050565b6001546001600160a01b03163314610b3c5760405162461bcd60e51b815260040161069e90612373565b6001600160a01b03808216600090815260186020908152604080832081516101008101835281549095168552600181015492850192909252600282015490840152600380820154606085015260048201546080850152600582015460a0850152600682015492939260c084019160ff90911690811115610bbe57610bbe61246f565b6003811115610bcf57610bcf61246f565b815260069190910154610100900460ff161515602090910152905060008160c001516003811115610c0257610c0261246f565b1415610c3c5760405162461bcd60e51b81526020600482015260096024820152681b9bdd08199bdd5b9960ba1b604482015260640161069e565b604081015115610c7b5760405162461bcd60e51b815260206004820152600a6024820152691a185cc81cdd185ad95960b21b604482015260640161069e565b610c86600b83611ee3565b506001600160a01b03821660008181526018602052604080822080546001600160a01b03191681556001810183905560028101839055600381018390556004810183905560058101839055600601805461ffff19169055517fe1434e25d6611e0db941968fdc97811c982ac1602e951637d206f5fdda9dd8f19190a25050565b6000610720600b83611ef8565b6001546001600160a01b03163314610d3d5760405162461bcd60e51b815260040161069e90612373565b6011805490829055604051829082907fb64c2fee5d0035d3aa1122935a0d2800f151a0853dd89adbabb795a6190f8be090600090a35050565b6001600160a01b03811660009081526018602052604090206003600682015460ff166003811115610da957610da961246f565b14610dc65760405162461bcd60e51b815260040161069e906123aa565b80546001600160a01b03163314610e0c5760405162461bcd60e51b815260206004820152600a60248201526937b7363c9037bbb732b960b11b604482015260640161069e565b8060010154610e19610f0e565b11610e565760405162461bcd60e51b815260206004820152600d60248201526c1cdd1a5b1b081a5b881a985a5b609a1b604482015260640161069e565b60068101805460ff1916600217905560006001820155610e77600b83611f1a565b50816001600160a01b03167f198b4f09d57ab5dbbf891a135940a04087b2544bebf3506cc81e1b64063b6d65610a2f610f0e565b6001546001600160a01b03163314610ed55760405162461bcd60e51b815260040161069e90612373565b6012805490829055604051829082907fdf736d7e2a17c66d20e9bd8c8b51ee5d59a97733dde732893d13fee45469f99b90600090a35050565b6000610f23610f1c600f5490565b4390611d57565b905090565b60026000541415610f7b5760405162461bcd60e51b815260206004820152601f60248201527f5265656e7472616e637947756172643a207265656e7472616e742063616c6c00604482015260640161069e565b6002600055601454811015610fbc5760405162461bcd60e51b8152602060048201526007602482015266746f6f206c6f7760c81b604482015260640161069e565b6009546040516323b872dd60e01b8152336004820152306024820152604481018390526001600160a01b03909116906323b872dd90606401602060405180830381600087803b15801561100e57600080fd5b505af1158015611022573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611046919061223f565b506001600160a01b038216600090815260186020526040812090600682015460ff1660038111156110795761107961246f565b146110b65760405162461bcd60e51b815260206004820152600d60248201526c105b1c9958591e48195e1a5cdd609a1b604482015260640161069e565b80546001600160a01b0319163317815560068101805460ff19166001179055600281018290556040516001600160a01b038416907fe366c1c0452ed8eec96861e9e54141ebff23c9ec89fe27b996b45f5ec388498790600090a25050600160005550565b6001546001600160a01b031633146111445760405162461bcd60e51b815260040161069e90612373565b600a80546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f3aaaebeb4821d6a7e5c77ece53cff0afcc56c82add2c978dbbb7f73e84cbcfd290600090a35050565b6015548110156111d25760405162461bcd60e51b8152602060048201526007602482015266746f6f206c6f7760c81b604482015260640161069e565b6009546040516323b872dd60e01b8152336004820152306024820152604481018390526001600160a01b03909116906323b872dd90606401602060405180830381600087803b15801561122457600080fd5b505af1158015611238573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061125c919061223f565b506001600160a01b03821660008181526019602090815260408083203384528252808320938352601890915290206002600682015460ff1660038111156112a5576112a561246f565b146112c25760405162461bcd60e51b815260040161069e906123aa565b8154156113055760006112f783600101546108cb64e8d4a510006108c586600401548860000154611d4490919063ffffffff16565b90506113033382611d6f565b505b61130d610f0e565b6002830155815461131e9084611f2f565b808355600482015461133b9164e8d4a51000916108c59190611d44565b6001830155600281015461134f9084611f2f565b60028201556008546113619084611f2f565b6008556001600160a01b0384166000908152600d602052604090206113869033611f1a565b506040518381526001600160a01b0385169033907f99039fcf0a98f484616c5196ee8b2ecfa971babf0b519848289ea4db381f85f79060200161093f565b6000610f2360016113d3610f0e565b90611f2f565b6001546001600160a01b031633146114035760405162461bcd60e51b815260040161069e90612373565b6001600160a01b03811660009081526018602052604090206001600682015460ff1660038111156114365761143661246f565b146114535760405162461bcd60e51b815260040161069e906123aa565b61145e600b83611f1a565b506006810180546002919060ff19166001836109e4565b6001546001600160a01b0316331461149f5760405162461bcd60e51b815260040161069e90612373565b6016805490829055604051829082907f79a5349732f93288abbb68e251c3dfc325bf3ee6fde7786d919155d39733e0f590600090a35050565b6001546001600160a01b031633146115025760405162461bcd60e51b815260040161069e90612373565b6010805490829055604051829082907fad65adbcfea0d9e94f88fee2e0422a61d519dc522506374972587aefe7194bd490600090a35050565b6001600160a01b038216600090815260196020908152604080832033845290915290208054821180159061156f5750600082115b6115b05760405162461bcd60e51b81526020600482015260126024820152716e6f7468696e6720746f20756e7374616b6560701b604482015260640161069e565b6115b8610f0e565b6115c982600201546113d360135490565b106116025760405162461bcd60e51b81526020600482015260096024820152686e6f7420616c6c6f7760b81b604482015260640161069e565b6001600160a01b038316600090815260186020526040812060018301546004820154845492939261164392916108cb9164e8d4a51000916108c59190611d44565b83549091506116529085611d63565b808455600483015461166f9164e8d4a51000916108c59190611d44565b600184015560028201546116839085611d63565b60028301556008546116959085611d63565b60085582546116c2576001600160a01b0385166000908152600d602052604090206116c09033611ee3565b505b6116d5336116d08684611f2f565b611d6f565b60408051858152602081018390526001600160a01b0387169133917f18edd09e80386cd99df397e2e0d87d2bb259423eae08645e776321a36fe680ef910160405180910390a35050505050565b4133146117715760405162461bcd60e51b815260206004820152601f60248201527f4f6e6c7920636f696e626173652063616e2063616c6c2066756e6374696f6e00604482015260640161069e565b6001600160a01b03811660009081526018602052604090206002600682015460ff1660038111156117a4576117a461246f565b146117dd5760405162461bcd60e51b81526020600482015260096024820152681b9bdd08199bdd5b9960ba1b604482015260640161069e565b60006117e7610f0e565b9050600061180360018460050154611f2f90919063ffffffff16565b60058401819055905061181560115490565b81141561188d57611829826113d360125490565b600184015560068301805460ff19166003179055611848600b85611ee3565b50836001600160a01b03167feb7d7a49847ec491969db21a0e31b234565a9923145a2d1b56a75c9e958258028360405161188491815260200190565b60405180910390a25b60408051828152602081018490526001600160a01b038616917f83b04ecf7330997e742429a641e136d9f3698c3e9ac9cb9ce0cc2d6da36a244d910160405180910390a250505050565b6060610f23600b611f3b565b606060008267ffffffffffffffff811115611900576119006124b1565b604051908082528060200260200182016040528015611929578160200160208202803683370190505b509050600061193786611c46565b905060005b848110156119c6576000611954826113d38989611d44565b9050828110156119b3576001600160a01b0388166000908152600d602052604090206119809082611f48565b8483815181106119925761199261249b565b60200260200101906001600160a01b031690816001600160a01b0316815250505b50806119be8161243e565b91505061193c565b509095945050505050565b6001546001600160a01b031633146119fb5760405162461bcd60e51b815260040161069e90612373565b6014805490829055604051829082907f207082661d623a88e041ad2d52c2d4ddc719880c70c3ab44aa81accff9bd86ed90600090a35050565b6001546001600160a01b03163314611a5e5760405162461bcd60e51b815260040161069e90612373565b6001600160a01b038116611ac35760405162461bcd60e51b815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201526564647265737360d01b606482015260840161069e565b611acc81611f54565b50565b413314611b1e5760405162461bcd60e51b815260206004820152601f60248201527f4f6e6c7920636f696e626173652063616e2063616c6c2066756e6374696f6e00604482015260640161069e565b6001600160a01b03811660009081526018602052604090206002600682015460ff166003811115611b5157611b5161246f565b14611b6e5760405162461bcd60e51b815260040161069e906123aa565b6000611b9282600201546108c564e8d4a51000601654611d4490919063ffffffff16565b6004830154909150611ba49082611f2f565b60048301556016546003830154611bba91611f2f565b6003830155601654601754611bce91611f2f565b601755601654604080519182524360208301526001600160a01b038516917f071464ed23d89f47be15656b77a0b638a220b7f85ba6bf9db649c41f64142a45910160405180910390a2505050565b60048181548110611c2c57600080fd5b6000918252602090912001546001600160a01b0316905081565b6001600160a01b0381166000908152600d6020526040812061072090611fa6565b6001546001600160a01b03163314611c915760405162461bcd60e51b815260040161069e90612373565b600f805490829055604051829082907f62d532a388a6e5e7ad8089a8aff169a6045b666b20a5a11070805ffc8ed16ee190600090a35050565b6001546001600160a01b03163314611cf45760405162461bcd60e51b815260040161069e90612373565b6001600160a01b03811660009081526018602052604090206003600682015460ff166003811115611d2757611d2761246f565b14610e565760405162461bcd60e51b815260040161069e906123aa565b6000611d508284612408565b9392505050565b6000611d5082846123e6565b6000611d508284612427565b6009546040516370a0823160e01b81523060048201526000916001600160a01b0316906370a082319060240160206040518083038186803b158015611db357600080fd5b505afa158015611dc7573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611deb919061227a565b90506000611e06601754600854611f2f90919063ffffffff16565b9050611e128184611f2f565b821015611e565760405162461bcd60e51b81526020600482015260126024820152716e6f7420656e6f7567682062616c616e636560701b604482015260640161069e565b60095460405163a9059cbb60e01b81526001600160a01b038681166004830152602482018690529091169063a9059cbb90604401602060405180830381600087803b158015611ea457600080fd5b505af1158015611eb8573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611edc919061223f565b5050505050565b6000611d50836001600160a01b038416611fb0565b6001600160a01b03811660009081526001830160205260408120541515611d50565b6000611d50836001600160a01b0384166120a3565b6000611d5082846123ce565b60606000611d50836120f2565b6000611d50838361214e565b600180546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b6000610720825490565b60008181526001830160205260408120548015612099576000611fd4600183612427565b8554909150600090611fe890600190612427565b905081811461204d5760008660000182815481106120085761200861249b565b906000526020600020015490508087600001848154811061202b5761202b61249b565b6000918252602080832090910192909255918252600188019052604090208390555b855486908061205e5761205e612485565b600190038181906000526020600020016000905590558560010160008681526020019081526020016000206000905560019350505050610720565b6000915050610720565b60008181526001830160205260408120546120ea57508154600181810184556000848152602080822090930184905584548482528286019093526040902091909155610720565b506000610720565b60608160000180548060200260200160405190810160405280929190818152602001828054801561214257602002820191906000526020600020905b81548152602001906001019080831161212e575b50505050509050919050565b60008260000182815481106121655761216561249b565b9060005260206000200154905092915050565b80356001600160a01b038116811461218f57600080fd5b919050565b6000602082840312156121a657600080fd5b611d5082612178565b600080604083850312156121c257600080fd5b6121cb83612178565b91506121d960208401612178565b90509250929050565b600080604083850312156121f557600080fd5b6121fe83612178565b946020939093013593505050565b60008060006060848603121561222157600080fd5b61222a84612178565b95602085013595506040909401359392505050565b60006020828403121561225157600080fd5b81518015158114611d5057600080fd5b60006020828403121561227357600080fd5b5035919050565b60006020828403121561228c57600080fd5b5051919050565b6000602082840312156122a557600080fd5b813563ffffffff81168114611d5057600080fd5b6001600160a01b03891681526020810188905260408101879052606081018690526080810185905260a0810184905261010081016004841061230b57634e487b7160e01b600052602160045260246000fd5b8360c083015282151560e08301529998505050505050505050565b6020808252825182820181905260009190848201906040850190845b818110156123675783516001600160a01b031683529284019291840191600101612342565b50909695505050505050565b6020808252601c908201527f4f6e6c79206f776e65722063616e2063616c6c2066756e6374696f6e00000000604082015260600190565b6020808252600a90820152696261642073746174757360b01b604082015260600190565b600082198211156123e1576123e1612459565b500190565b60008261240357634e487b7160e01b600052601260045260246000fd5b500490565b600081600019048311821515161561242257612422612459565b500290565b60008282101561243957612439612459565b500390565b600060001982141561245257612452612459565b5060010190565b634e487b7160e01b600052601160045260246000fd5b634e487b7160e01b600052602160045260246000fd5b634e487b7160e01b600052603160045260246000fd5b634e487b7160e01b600052603260045260246000fd5b634e487b7160e01b600052604160045260246000fdfea2646970667358221220037af03eb50adcf840ebcf1c9c0a1d56c4e83c3fca5c769d522db630ef2305dc64736f6c63430008060033",
			},
		},
	}
}

func UpgradeSystem(
	chainID int,
	config *chain.Forks,
	blockNumber uint64,
	txn *state.Txn,
	logger hclog.Logger,
) {
	if config == nil || blockNumber == 0 || txn == nil {
		return
	}

	var network string

	switch chainID {
	case MainNetChainID:
		fallthrough
	default:
		network = mainNet
	}

	// only upgrade portland once
	if config.IsOnPortland(blockNumber) {
		up := _portlandUpgrade[network]
		applySystemContractUpgrade(up, blockNumber, txn,
			logger.With("upgrade", up.UpgradeName, "network", network))
	}

	// only upgrade detroit once
	if config.IsOnDetroit(blockNumber) {
		up := _detroitUpgrade[network]
		applySystemContractUpgrade(up, blockNumber, txn,
			logger.With("upgrade", up.UpgradeName, "network", network))
	}
}

func applySystemContractUpgrade(upgrade *Upgrade, blockNumber uint64, txn *state.Txn, logger hclog.Logger) {
	if upgrade == nil {
		logger.Info("Empty upgrade config", "height", blockNumber)

		return
	}

	logger.Info(fmt.Sprintf("Apply upgrade %s at height %d", upgrade.UpgradeName, blockNumber))

	for _, cfg := range upgrade.Configs {
		logger.Info(fmt.Sprintf("Upgrade contract %s to commit %s", cfg.ContractAddr.String(), cfg.CommitURL))

		newContractCode, err := hex.DecodeHex(cfg.Code)
		if err != nil {
			panic(fmt.Errorf("failed to decode new contract code: %w", err))
		}

		txn.SetCode(cfg.ContractAddr, newContractCode)
	}
}
