var Web3 = require('web3')
var web3 = new Web3(new Web3.providers.HttpProvider("http://127.0.0.1:8545"));

var _greeting = "Greeting one" ;
var greeterAddress
var greeterContract = web3.eth.contract([{"constant":false,"inputs":[{"name":"_greeting","type":"string"}],"name":"setGreeting","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"greet","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_greeting","type":"string"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]);
var greeter = greeterContract.new(
   _greeting,
   {
     from: web3.eth.accounts[0], 
     data: '0x608060405234801561001057600080fd5b5060405161041c38038061041c833981018060405281019080805182019291905050508060009080519060200190610049929190610050565b50506100f5565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061009157805160ff19168380011785556100bf565b828001600101855582156100bf579182015b828111156100be5782518255916020019190600101906100a3565b5b5090506100cc91906100d0565b5090565b6100f291905b808211156100ee5760008160009055506001016100d6565b5090565b90565b610318806101046000396000f30060806040526004361061004c576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063a413686214610051578063cfae3217146100ba575b600080fd5b34801561005d57600080fd5b506100b8600480360381019080803590602001908201803590602001908080601f016020809104026020016040519081016040528093929190818152602001838380828437820191505050505050919291929050505061014a565b005b3480156100c657600080fd5b506100cf6101a5565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561010f5780820151818401526020810190506100f4565b50505050905090810190601f16801561013c5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b33600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555080600090805190602001906101a1929190610247565b5050565b606060008054600181600116156101000203166002900480601f01602080910402602001604051908101604052809291908181526020018280546001816001161561010002031660029004801561023d5780601f106102125761010080835404028352916020019161023d565b820191906000526020600020905b81548152906001019060200180831161022057829003601f168201915b5050505050905090565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061028857805160ff19168380011785556102b6565b828001600101855582156102b6579182015b828111156102b557825182559160200191906001019061029a565b5b5090506102c391906102c7565b5090565b6102e991905b808211156102e55760008160009055506001016102cd565b5090565b905600a165627a7a7230582006a0234c8e31f8b96b20b9fb2f0ddddde4463524c8091978b490c87277170b620029', 
     gas: '4700000'
   }, function (e, contract){
    if (typeof contract.address !== 'undefined') {
      var _address = contract.address
      var proxygreeterContract = web3.eth.contract([{"constant":false,"inputs":[{"name":"_greeting","type":"string"}],"name":"proxySetGreeting","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"inputs":[{"name":"_address","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]);
      var proxygreeter = proxygreeterContract.new(
         _address,
         {
           from: web3.eth.accounts[0], 
           data: '0x608060405234801561001057600080fd5b5060405160208061026e83398101806040528101908080519060200190929190505050806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550506101eb806100836000396000f300608060405260043610610041576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806369c92f6a14610046575b600080fd5b34801561005257600080fd5b506100ad600480360381019080803590602001908201803590602001908080601f01602080910402602001604051908101604052809392919081815260200183838082843782019150505050505091929192905050506100af565b005b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663a4136862826040518263ffffffff167c01000000000000000000000000000000000000000000000000000000000281526004018080602001828103825283818151815260200191508051906020019080838360005b8381101561015857808201518184015260208101905061013d565b50505050905090810190601f1680156101855780820380516001836020036101000a031916815260200191505b5092505050600060405180830381600087803b1580156101a457600080fd5b505af11580156101b8573d6000803e3d6000fd5b50505050505600a165627a7a72305820946d166088750797f5bc3f5953806fd10118c88c95eac53d7aa7628997b69a310029', 
           gas: '4700000'
         }, function (e, contract){
          if (typeof contract.address !== 'undefined') {
               console.log('Proxy Greeter Contract mined! address: ' + contract.address + ' transactionHash: ' + contract.transactionHash);
          }
       })
      console.log('Greeter Contract mined! address: ' + contract.address + ' transactionHash: ' + contract.transactionHash);
    }
 })