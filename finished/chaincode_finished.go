/*
Copyright IBM Corp 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// AssetObject struct
type AssetObject struct {
	Serialno string
	Partno   string
	Owner    string
}

var aucTables = []string{"AssetTable", "ContractHistory"}

func GetNumberOfKeys(tname string) int {
	TableMap := map[string]int{
		"AssetTable":      1,
		"ContractHistory": 3,
	}
	return TableMap[tname]
}

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SimpleChaincode - Init Chaincode implementation - The following sequence of transactions can be used to test the Chaincode
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	// TODO - Include all initialization to be complete before Invoke and Query
	// Uses aucTables to delete tables if they exist and re-create them
	//myLogger.Info("[Trade and Auction Application] Init")
	fmt.Println("[Trade and Auction Application] Init")
	var err error

	for _, val := range aucTables {
		err = stub.DeleteTable(val)
		if err != nil {
			return nil, fmt.Errorf("Init(): DeleteTable of %s  Failed ", val)
		}
		err = InitLedger(stub, val)
		if err != nil {
			return nil, fmt.Errorf("Init(): InitLedger of %s  Failed ", val)
		}
	}
	// Update the ledger with the Application version
	err = stub.PutState("version", []byte(strconv.Itoa(23)))
	if err != nil {
		return nil, err
	}

	fmt.Println("Init() Initialization Complete  : ", args)
	return []byte("Init(): Initialization Complete"), nil
}

func InitLedger(stub shim.ChaincodeStubInterface, tableName string) error {

	// Generic Table Creation Function - requires Table Name and Table Key Entry
	// Create Table - Get number of Keys the tables supports
	// This version assumes all Keys are String and the Data is Bytes
	// This Function can replace all other InitLedger function in this app such as InitItemLedger()

	nKeys := GetNumberOfKeys(tableName)
	if nKeys < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
		fmt.Println("Auction_Application: Failed creating Table ", tableName)
		return errors.New("Auction_Application: Failed creating Table " + tableName)
	}

	var columnDefsForTbl []*shim.ColumnDefinition

	for i := 0; i < nKeys; i++ {
		columnDef := shim.ColumnDefinition{Name: "keyName" + strconv.Itoa(i), Type: shim.ColumnDefinition_STRING, Key: true}
		columnDefsForTbl = append(columnDefsForTbl, &columnDef)
	}

	columnLastTblDef := shim.ColumnDefinition{Name: "Details", Type: shim.ColumnDefinition_BYTES, Key: false}
	columnDefsForTbl = append(columnDefsForTbl, &columnLastTblDef)

	// Create the Table (Nil is returned if the Table exists or if the table is created successfully
	err := stub.CreateTable(tableName, columnDefsForTbl)

	if err != nil {
		fmt.Println("Auction_Application: Failed creating Table ", tableName)
		return errors.New("Auction_Application: Failed creating Table " + tableName)
	}

	return err
}

// Invoke is our entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "postAsset" {
		return t.PostAsset(stub, args)
	}
	fmt.Println("invoke did not find func: " + function) //error
	return nil, errors.New("Received unknown function invocation: " + function)
}

// Invoke is our entry point to invoke a chaincode function
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "getAsset" {
		return t.GetAsset(stub, args)
	} else if function == "getHistory" {
		return t.GetHistory(stub, args)
	}
	fmt.Println("invoke did not find func: " + function) //error

	return nil, errors.New("Received unknown function invocation: " + function)
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Create a master Object of the Item
// Since the Owner Changes hands, a record has to be written for each
// Transaction with the updated Encryption Key of the new owner
// Example
//./peer chaincode invoke -l golang -n mycc -c '{"Function": "PostItem", "Args":["1000", "ARTINV", "Shadows by Asppen", "Asppen Messer", "20140202", "Original", "Landscape" , "Canvas", "15 x 15 in", "sample_7.png","$600", "100"]}'
/////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (t *SimpleChaincode) PostAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	assetObject, err := CreateAsset(args[0:])
	if err != nil {
		fmt.Println("PostItem(): Cannot create item object \n")
		return nil, err
	}

	/*// Check if the Owner ID specified is registered and valid
	ownerInfo, err := ValidateMember(stub, itemObject.CurrentOwnerID)
	fmt.Println("Owner information  ", ownerInfo, itemObject.CurrentOwnerID)
	if err != nil {
		fmt.Println("PostItem() : Failed Owner information not found for ", itemObject.CurrentOwnerID)
		return nil, err
	}*/

	// Convert Item Object to JSON
	buff, err := ARtoJSON(assetObject) //
	if err != nil {
		fmt.Println("PostItem() : Failed Cannot create object buffer for write : ", args[1])
		return nil, errors.New("PostItem(): Failed Cannot create object buffer for write : " + args[1])
	} else {
		// Update the ledger with the Buffer Data
		// err = stub.PutState(args[0], buff)
		keys := []string{args[0]}
		err = UpdateLedger(stub, "AssetTable", keys, buff)
		if err != nil {
			fmt.Println("PostItem() : write error while inserting record\n")
			return buff, err
		}
		//update Asset History table
		keys1 := []string{"2016", assetObject.Serialno, "random"}
		err = UpdateLedger(stub, "ContractHistory", keys1, buff)
		if err != nil {
			fmt.Println("PostItem() : write error while inserting record into contract history table\n")
			return buff, err
		}
		return nil, nil
	}
}

// CreateAssetObject creates an asset
func CreateAsset(args []string) (AssetObject, error) {
	// S001 LHTMO bosch
	var err error
	var myAsset AssetObject

	// Check there are 3 Arguments provided as per the the struct
	if len(args) != 3 {
		fmt.Println("CreateAssetObject(): Incorrect number of arguments. Expecting 3 ")
		return myAsset, errors.New("CreateAssetObject(): Incorrect number of arguments. Expecting 3 ")
	}

	// Validate Serialno is an integer

	_, err = strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("CreateAssetObject(): SerialNo should be an integer create failed! ")
		return myAsset, errors.New("CreateAssetbject(): SerialNo should be an integer create failed. ")
	}

	myAsset = AssetObject{args[0], args[1], args[2]}

	fmt.Println("CreateAssetObject(): Asset Object created: ", myAsset.Serialno, myAsset.Partno, myAsset.Owner)
	return myAsset, nil
}

// ARtoJSON Converts an Asset Object to a JSON String
func ARtoJSON(ast AssetObject) ([]byte, error) {

	ajson, err := json.Marshal(ast)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return ajson, nil
}

func UpdateLedger(stub shim.ChaincodeStubInterface, tableName string, keys []string, args []byte) error {

	nKeys := GetNumberOfKeys(tableName)
	if nKeys < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
	}

	var columns []*shim.Column

	for i := 0; i < nKeys; i++ {
		col := shim.Column{Value: &shim.Column_String_{String_: keys[i]}}
		columns = append(columns, &col)
	}

	lastCol := shim.Column{Value: &shim.Column_Bytes{Bytes: []byte(args)}}
	columns = append(columns, &lastCol)

	row := shim.Row{columns}
	ok, err := stub.InsertRow(tableName, row)
	if err != nil {
		return fmt.Errorf("UpdateLedger: InsertRow into "+tableName+" Table operation failed. %s", err)
	}
	if !ok {
		return errors.New("UpdateLedger: InsertRow into " + tableName + " Table failed. Row with given key " + keys[0] + " already exists")
	}

	fmt.Println("UpdateLedger: InsertRow into ", tableName, " Table operation Successful. ")
	return nil
}

func (t *SimpleChaincode) GetAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var err error

	// Get the Objects and Display it
	Avalbytes, err := QueryLedger(stub, "AssetTable", args)

	if err != nil {
		fmt.Println("GetItem() : Failed to Query Object ")
		jsonResp := "{\"Error\":\"Failed to get  Object Data for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	if Avalbytes == nil {
		fmt.Println("GetItem() : Incomplete Query Object ")
		jsonResp := "{\"Error\":\"Incomplete information about the key for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	fmt.Println("GetItem() : Response : Successfull ")
	return Avalbytes, nil
}

func JSONtoAR(data []byte) (AssetObject, error) {

	ar := AssetObject{}
	err := json.Unmarshal([]byte(data), &ar)
	if err != nil {
		fmt.Println("Unmarshal failed : ", err)
	}

	return ar, err
}

//random comment to checkin

func QueryLedger(stub shim.ChaincodeStubInterface, tableName string, args []string) ([]byte, error) {

	var columns []shim.Column
	nCol := GetNumberOfKeys(tableName)
	for i := 0; i < nCol; i++ {
		colNext := shim.Column{Value: &shim.Column_String_{String_: args[i]}}
		columns = append(columns, colNext)
	}

	row, err := stub.GetRow(tableName, columns)
	if err != nil {
		return nil, errors.New("unable to query row!")
	}
	fmt.Println("Length or number of rows retrieved ", len(row.Columns))

	if len(row.Columns) == 0 {
		jsonResp := "{\"Error\":\"Failed retrieving data " + args[0] + ". \"}"
		fmt.Println("Error retrieving data record for Key = ", args[0], "Error : ", jsonResp)
		return nil, errors.New(jsonResp)
	}

	//fmt.Println("User Query Response:", row)
	//jsonResp := "{\"Owner\":\"" + string(row.Columns[nCol].GetBytes()) + "\"}"
	//fmt.Println("User Query Response:%s\n", jsonResp)
	Avalbytes := row.Columns[nCol].GetBytes()

	// Perform Any additional processing of data
	fmt.Println("QueryLedger() : Successful - Proceeding to ProcessRequestType ")
	return Avalbytes, nil
}

func (t *SimpleChaincode) GetHistory(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var err error

	// Get the Objects and Display it
	Avalbytes, err := QueryLedger(stub, "ContractHistory", args)

	if err != nil {
		fmt.Println("GetItem() : Failed to Query Object ")
		jsonResp := "{\"Error\":\"Failed to get  Object Data for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	if Avalbytes == nil {
		fmt.Println("GetItem() : Incomplete Query Object ")
		jsonResp := "{\"Error\":\"Incomplete information about the key for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	fmt.Println("GetItem() : Response : Successfull ")
	return Avalbytes, nil
}

func GetListOfContractHistory(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	rows, err := GetList(stub, "ContractHistory", args)
	if err != nil {
		return nil, fmt.Errorf("GetListOfOpenAucs operation failed. Error marshaling JSON: %s", err)
	}

	nCol := GetNumberOfKeys("ContractHistory")

	tlist := make([]AssetObject, len(rows))
	for i := 0; i < len(rows); i++ {
		ts := rows[i].Columns[nCol].GetBytes()
		ar, err := JSONtoAR(ts)
		if err != nil {
			fmt.Println("GetListOfOpenAucs() Failed : Ummarshall error")
			return nil, fmt.Errorf("GetListOfOpenAucs() operation failed. %s", err)
		}
		tlist[i] = ar
	}

	jsonRows, _ := json.Marshal(tlist)

	//fmt.Println("List of Open Auctions : ", jsonRows)
	return jsonRows, nil

}

func GetList(stub shim.ChaincodeStubInterface, tableName string, args []string) ([]shim.Row, error) {
	var columns []shim.Column

	nKeys := GetNumberOfKeys(tableName)
	nCol := len(args)
	if nCol < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
		return nil, errors.New("GetList failed. Must include at least key values")
	}

	for i := 0; i < nCol; i++ {
		colNext := shim.Column{Value: &shim.Column_String_{String_: args[i]}}
		columns = append(columns, colNext)
	}

	rowChannel, err := stub.GetRows(tableName, columns)
	if err != nil {
		return nil, fmt.Errorf("GetList operation failed. %s", err)
	}
	var rows []shim.Row
	for {
		select {
		case row, ok := <-rowChannel:
			if !ok {
				rowChannel = nil
			} else {
				rows = append(rows, row)
				//If required enable for debugging
				//fmt.Println(row)
			}
		}
		if rowChannel == nil {
			break
		}
	}

	fmt.Println("Number of Keys retrieved : ", nKeys)
	fmt.Println("Number of rows retrieved : ", len(rows))
	return rows, nil
}
