

package core

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/fafdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis.go
//go:generate gencodec -type GenesisAccount -field-override genesisAccountMarshaling -out gen_genesis_account.go

var errGenesisNoConfig = errors.New("genesis has no chain configuration")

// Genesis specifies the header fields, state of a genesis block. It also defines hard
// fork switch-over blocks through the chain configuration.
type Genesis struct {
	Config     *params.ChainConfig `json:"config"`
	Nonce      uint64              `json:"nonce"`
	Timestamp  uint64              `json:"timestamp"`
	ExtraData  []byte              `json:"extraData"`
	GasLimit   uint64              `json:"gasLimit"   gencodec:"required"`
	Difficulty *big.Int            `json:"difficulty" gencodec:"required"`
	Mixhash    common.Hash         `json:"mixHash"`
	Coinbase   common.Address      `json:"coinbase"`
	Alloc      GenesisAlloc        `json:"alloc"      gencodec:"required"`

	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	Number     uint64      `json:"number"`
	GasUsed    uint64      `json:"gasUsed"`
	ParentHash common.Hash `json:"parentHash"`
}

// GenesisAlloc specifies the initial state that is part of the genesis block.
type GenesisAlloc map[common.Address]GenesisAccount

func (ga *GenesisAlloc) UnmarshalJSON(data []byte) error {
	m := make(map[common.UnprefixedAddress]GenesisAccount)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*ga = make(GenesisAlloc)
	for addr, a := range m {
		(*ga)[common.Address(addr)] = a
	}
	return nil
}

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Code       []byte                      `json:"code,omitempty"`
	Storage    map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance    *big.Int                    `json:"balance" gencodec:"required"`
	Nonce      uint64                      `json:"nonce,omitempty"`
	PrivateKey []byte                      `json:"secretKey,omitempty"` // for tests
}

// field type overrides for gencodec
type genesisSpecMarshaling struct {
	Nonce      math.HexOrDecimal64
	Timestamp  math.HexOrDecimal64
	ExtraData  hexutil.Bytes
	GasLimit   math.HexOrDecimal64
	GasUsed    math.HexOrDecimal64
	Number     math.HexOrDecimal64
	Difficulty *math.HexOrDecimal256
	Alloc      map[common.UnprefixedAddress]GenesisAccount
}

type genesisAccountMarshaling struct {
	Code       hexutil.Bytes
	Balance    *math.HexOrDecimal256
	Nonce      math.HexOrDecimal64
	Storage    map[storageJSON]storageJSON
	PrivateKey hexutil.Bytes
}

// storageJSON represents a 256 bit byte array, but allows less than 256 bits when
// unmarshaling from hex.
type storageJSON common.Hash

func (h *storageJSON) UnmarshalText(text []byte) error {
	text = bytes.TrimPrefix(text, []byte("0x"))
	if len(text) > 64 {
		return fmt.Errorf("too many hex characters in storage key/value %q", text)
	}
	offset := len(h) - len(text)/2 // pad on the left
	if _, err := hex.Decode(h[offset:], text); err != nil {
		fmt.Println(err)
		return fmt.Errorf("invalid hex storage key/value %q", text)
	}
	return nil
}

func (h storageJSON) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// GenesisMismatchError is raised when trying to overwrite an existing
// genesis block with an incompatible one.
type GenesisMismatchError struct {
	Stored, New common.Hash
}

func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database contains incompatible genesis (have %x, new %x)", e.Stored, e.New)
}

// SetupGenesisBlock writes or updates the genesis block in db.
// The block that will be used is:
//
//                          genesis == nil       genesis != nil
//                       +------------------------------------------
//     db has no genesis |  main-net default  |  genesis
//     db has genesis    |  from DB           |  genesis (if compatible)
//
// The stored chain configuration will be updated if it is compatible (i.e. does not
// specify a fork block below the local head block). In case of a conflict, the
// error is a *params.ConfigCompatError and the new, unwritten config is returned.
//
// The returned chain configuration is never nil.
func SetupGenesisBlock(db fafdb.Database, genesis *Genesis) (*params.ChainConfig, common.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return params.AllEthashProtocolChanges, common.Hash{}, errGenesisNoConfig
	}
	// Just commit the new block if there is no stored genesis block.
	stored := rawdb.ReadCanonicalHash(db, 0)
	if (stored == common.Hash{}) {
		if genesis == nil {
			log.Info("Writing default main-net genesis block")
			genesis = DefaultGenesisBlock()
		} else {
			log.Info("Writing custom genesis block")
		}
		block, err := genesis.Commit(db)
		if err != nil {
			return genesis.Config, common.Hash{}, err
		}
		return genesis.Config, block.Hash(), nil
	}

	// We have the genesis block in database(perhaps in ancient database)
	// but the corresponding state is missing.
	header := rawdb.ReadHeader(db, stored, 0)
	if _, err := state.New(header.Root, state.NewDatabaseWithConfig(db, nil), nil); err != nil {
		if genesis == nil {
			genesis = DefaultGenesisBlock()
		}
		// Ensure the stored genesis matches with the given one.
		hash := genesis.ToBlock(nil).Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
		block, err := genesis.Commit(db)
		if err != nil {
			return genesis.Config, hash, err
		}
		return genesis.Config, block.Hash(), nil
	}

	// Check whether the genesis block is already written.
	if genesis != nil {
		hash := genesis.ToBlock(nil).Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
	}

	// Get the existing chain configuration.
	newcfg := genesis.configOrDefault(stored)
	if err := newcfg.CheckConfigForkOrder(); err != nil {
		return newcfg, common.Hash{}, err
	}
	storedcfg := rawdb.ReadChainConfig(db, stored)
	if storedcfg == nil {
		log.Warn("Found genesis block without chain config")
		rawdb.WriteChainConfig(db, stored, newcfg)
		return newcfg, stored, nil
	}
	// Special case: don't change the existing config of a non-mainnet chain if no new
	// config is supplied. These chains would get AllProtocolChanges (and a compat error)
	// if we just continued here.
	if genesis == nil && stored != params.MainnetGenesisHash {
		return storedcfg, stored, nil
	}

	// Check config compatibility and write the config. Compatibility errors
	// are returned to the caller unless we're already at block zero.
	height := rawdb.ReadHeaderNumber(db, rawdb.ReadHeadHeaderHash(db))
	if height == nil {
		return newcfg, stored, fmt.Errorf("missing block number for head header hash")
	}
	compatErr := storedcfg.CheckCompatible(newcfg, *height)
	if compatErr != nil && *height != 0 && compatErr.RewindTo != 0 {
		return newcfg, stored, compatErr
	}
	rawdb.WriteChainConfig(db, stored, newcfg)
	return newcfg, stored, nil
}

func (g *Genesis) configOrDefault(ghash common.Hash) *params.ChainConfig {
	switch {
	case g != nil:
		return g.Config
	case ghash == params.MainnetGenesisHash:
		return params.MainnetChainConfig
	case ghash == params.RopstenGenesisHash:
		return params.RopstenChainConfig
	case ghash == params.RinkebyGenesisHash:
		return params.RinkebyChainConfig
	case ghash == params.GoerliGenesisHash:
		return params.GoerliChainConfig
	case ghash == params.YoloV2GenesisHash:
		return params.YoloV2ChainConfig
	default:
		return params.AllEthashProtocolChanges
	}
}

// ToBlock creates the genesis block and writes state of a genesis specification
// to the given database (or discards it if nil).
func (g *Genesis) ToBlock(db fafdb.Database) *types.Block {
	if db == nil {
		db = rawdb.NewMemoryDatabase()
	}
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(db), nil)
	for addr, account := range g.Alloc {
		statedb.AddBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}
	root := statedb.IntermediateRoot(false)
	head := &types.Header{
		Number:     new(big.Int).SetUint64(g.Number),
		Nonce:      types.EncodeNonce(g.Nonce),
		Time:       g.Timestamp,
		ParentHash: g.ParentHash,
		Extra:      g.ExtraData,
		GasLimit:   g.GasLimit,
		GasUsed:    g.GasUsed,
		Difficulty: g.Difficulty,
		MixDigest:  g.Mixhash,
		Coinbase:   g.Coinbase,
		Root:       root,
	}
	if g.GasLimit == 0 {
		head.GasLimit = params.GenesisGasLimit
	}
	if g.Difficulty == nil {
		head.Difficulty = params.GenesisDifficulty
	}
	statedb.Commit(false)
	statedb.Database().TrieDB().Commit(root, true, nil)

	return types.NewBlock(head, nil, nil, nil, new(trie.Trie))
}

// Commit writes the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func (g *Genesis) Commit(db fafdb.Database) (*types.Block, error) {
	block := g.ToBlock(db)
	if block.Number().Sign() != 0 {
		return nil, fmt.Errorf("can't commit genesis block with number > 0")
	}
	config := g.Config
	if config == nil {
		config = params.AllEthashProtocolChanges
	}
	if err := config.CheckConfigForkOrder(); err != nil {
		return nil, err
	}
	rawdb.WriteTd(db, block.Hash(), block.NumberU64(), g.Difficulty)
	rawdb.WriteBlock(db, block)
	rawdb.WriteReceipts(db, block.Hash(), block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(db, block.Hash())
	rawdb.WriteHeadFastBlockHash(db, block.Hash())
	rawdb.WriteHeadHeaderHash(db, block.Hash())
	rawdb.WriteChainConfig(db, block.Hash(), config)
	return block, nil
}

// MustCommit writes the genesis block and state to db, panicking on error.
// The block is committed as the canonical head block.
func (g *Genesis) MustCommit(db fafdb.Database) *types.Block {
	block, err := g.Commit(db)
	if err != nil {
		panic(err)
	}
	return block
}

// GenesisBlockForTesting creates and writes a block in which addr has the given wei balance.
func GenesisBlockForTesting(db fafdb.Database, addr common.Address, balance *big.Int) *types.Block {
	g := Genesis{Alloc: GenesisAlloc{addr: {Balance: balance}}}
	return g.MustCommit(db)
}

// 默认创世区块.
func DefaultGenesisBlock() *Genesis {
	//bigInt:=new(big.Int)
	//balance,_:=bigInt.SetString(`100000000000000000000000000`,10)
	//fmt.Println(balance,`balance`)
	return &Genesis{
		Config:     params.MainnetChainConfig,
		Nonce:      66,
		ExtraData:  hexutil.MustDecode("0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa"),
		GasLimit:   4294967295,
		Difficulty: big.NewInt(9869184),
		Alloc:     map[common.Address]GenesisAccount{
			common.HexToAddress(`fx0000000000000000000000000000000000000000`):{Balance:new(big.Int).SetStrings(`22810875592000000229376`,10)},
			common.HexToAddress(`fx003964787cb7873d71e83d9335750a901e8ffd16`):{Balance:new(big.Int).SetStrings(`4663900000000000000`,10)},
			common.HexToAddress(`fx007a55cd21a1436e4a913ddf8406a9e0152b3f4f`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx00d93a1d3bb76ab1cd6640485b0316bb1579a386`):{Balance:new(big.Int).SetStrings(`43905800000000000000`,10)},
			common.HexToAddress(`fx014e83947024c7ea54e6b4a9dc2b5f60831fba1a`):{Balance:new(big.Int).SetStrings(`6844900000000000000`,10)},
			common.HexToAddress(`fx0221a0e61dce334c19d3e22c76dd46add078853b`):{Balance:new(big.Int).SetStrings(`97578500000000000000`,10)},
			common.HexToAddress(`fx02876689c32784c379bde753c66600d0ba99d964`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fx04e9078d306379bd6d6752889e7a56d1047f0188`):{Balance:new(big.Int).SetStrings(`15948000000000000`,10)},
			common.HexToAddress(`fx04eb375d216691600d12350dc86a1ca4f111cd32`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx04fae7ce4c53a92abd6e9490c7826cf753f6735c`):{Balance:new(big.Int).SetStrings(`14039537000000000000`,10)},
			common.HexToAddress(`fx056360f9c3d97b27fe28f252fa0f617ec8d7b3fd`):{Balance:new(big.Int).SetStrings(`469000000000000`,10)},
			common.HexToAddress(`fx05d01c82c535d187663371eae5d68ae08ec7421c`):{Balance:new(big.Int).SetStrings(`1120379000000000000`,10)},
			common.HexToAddress(`fx0874c4040d1e0454b2ff834eb9fa519718c0c451`):{Balance:new(big.Int).SetStrings(`24265800000000000000`,10)},
			common.HexToAddress(`fx097c750290a98c334b87095e0981c883be87f41d`):{Balance:new(big.Int).SetStrings(`42526279000000004096`,10)},
			common.HexToAddress(`fx0ac1b7664a1bd4fba45ffdde31ffbfc867d455e7`):{Balance:new(big.Int).SetStrings(`2422979000000000000`,10)},
			common.HexToAddress(`fx0ace0596c3c5e0f3c4d6f6180aaa15f4888bdab5`):{Balance:new(big.Int).SetStrings(`69485406004487094272`,10)},
			common.HexToAddress(`fx0ae46bab374355d6e59e3c7ff69c9463f917b9e5`):{Balance:new(big.Int).SetStrings(`7456300000000000000`,10)},
			common.HexToAddress(`fx0d432f0118fd48170086ae5d5a4fd0f3caaea984`):{Balance:new(big.Int).SetStrings(`123837300000000000000`,10)},
			common.HexToAddress(`fx0d690f7e2efde2fce9ed518ee9ff3fcaa7ee118a`):{Balance:new(big.Int).SetStrings(`1321116000000000000`,10)},
			common.HexToAddress(`fx0dcbb89184a60dc9e650e6ace300de43ad6332f9`):{Balance:new(big.Int).SetStrings(`241987695000000004096`,10)},
			common.HexToAddress(`fx0e0f56e778f49f3e27f98ea09b7b6884606db22e`):{Balance:new(big.Int).SetStrings(`830674000000000000`,10)},
			common.HexToAddress(`fx0fb832292928619b7b6ba688fe09ad3524b4a549`):{Balance:new(big.Int).SetStrings(`2094931694999999873024`,10)},
			common.HexToAddress(`fx10738fbf7ec1300a9cc87b375751dd93af2b072b`):{Balance:new(big.Int).SetStrings(`185701000000000000`,10)},
			common.HexToAddress(`fx10b630364122935217db7a0b1477939afdc95b7b`):{Balance:new(big.Int).SetStrings(`48472300000000000000`,10)},
			common.HexToAddress(`fx10d7c46c9941cf0328e7a916cc88723c17888d14`):{Balance:new(big.Int).SetStrings(`5222579000000000000`,10)},
			common.HexToAddress(`fx10f7eab22d94567d6d0d42c1304eb988cff68e45`):{Balance:new(big.Int).SetStrings(`5480400000000000000`,10)},
			common.HexToAddress(`fx1125cb17fe975ce2830cabf6313a822c8cee394a`):{Balance:new(big.Int).SetStrings(`7059537000000000000`,10)},
			common.HexToAddress(`fx112783a6daeb98e61e9f5bfebe789d3f9271b827`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx12c9892ceafe6af2a4cce043e89a9079292b8a81`):{Balance:new(big.Int).SetStrings(`6045600000000000000`,10)},
			common.HexToAddress(`fx13e4bc3d575a234ec92fb766db815d6bccd7c86e`):{Balance:new(big.Int).SetStrings(`244730679000000004096`,10)},
			common.HexToAddress(`fx140fd03e399be852d00499659ab381865cb9b919`):{Balance:new(big.Int).SetStrings(`17356500000000000000`,10)},
			common.HexToAddress(`fx1469ce6fb15cdb401bb3d43d8f9145f46da99a2a`):{Balance:new(big.Int).SetStrings(`569581999999983616`,10)},
			common.HexToAddress(`fx15140a01c66da74101204b8d87078fe11568e162`):{Balance:new(big.Int).SetStrings(`2265758000000000000`,10)},
			common.HexToAddress(`fx1572ac1bef72cbb31d9a58d5ba3832cad2b58639`):{Balance:new(big.Int).SetStrings(`64550000000000000000`,10)},
			common.HexToAddress(`fx15d370cdae70ccaeeaec19d57fe14428b71dedc9`):{Balance:new(big.Int).SetStrings(`26415800000000000000`,10)},
			common.HexToAddress(`fx175d462ffc7c1be90b0eafd542be93df50e664bc`):{Balance:new(big.Int).SetStrings(`31772679000000000000`,10)},
			common.HexToAddress(`fx176de57a3fa5c3e3d01fcf460c9a301be090e8f3`):{Balance:new(big.Int).SetStrings(`1000000000000000000`,10)},
			common.HexToAddress(`fx17bac44bbaccadda47fbf3e92eba5a6212d70bcc`):{Balance:new(big.Int).SetStrings(`107911700000000000000`,10)},
			common.HexToAddress(`fx1857d7a64c6f3424d4c5d5ec21e64357050f6236`):{Balance:new(big.Int).SetStrings(`5480400000000000000`,10)},
			common.HexToAddress(`fx1885c5a0be23e1e5eba33b1da3501c920a36ea16`):{Balance:new(big.Int).SetStrings(`24265800000000000000`,10)},
			common.HexToAddress(`fx18f8fcc37317b4a43b16229b7977a22c30bc2fa4`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fx19a3a69116cfb03344af4e3302b9062c40611bf7`):{Balance:new(big.Int).SetStrings(`37544600000000000000`,10)},
			common.HexToAddress(`fx1a9f24b6af27170c2ac53367d27f98b283cf0917`):{Balance:new(big.Int).SetStrings(`166416400000000000000`,10)},
			common.HexToAddress(`fx1af0b4accd4794d8e0cfc9ee9488e0368a6b31b0`):{Balance:new(big.Int).SetStrings(`2727479000000000000`,10)},
			common.HexToAddress(`fx1bb2dcd2ad8c0cbf566d2061d5acaa9c4005657f`):{Balance:new(big.Int).SetStrings(`238763800000000000000`,10)},
			common.HexToAddress(`fx1d35b9326b055153dbe5ddc46f4ee0d9b2269392`):{Balance:new(big.Int).SetStrings(`4627833000000032768`,10)},
			common.HexToAddress(`fx1e540a4e5e174e938eadc18667c62e217b46289c`):{Balance:new(big.Int).SetStrings(`258125778999999987712`,10)},
			common.HexToAddress(`fx1f2de1bd18e36eca1898ac5a125bd78bb1b56582`):{Balance:new(big.Int).SetStrings(`105585516000000000000`,10)},
			common.HexToAddress(`fx1f4f495e55786a02fcfb7a1bbee59c684a379e82`):{Balance:new(big.Int).SetStrings(`48402158000000008192`,10)},
			common.HexToAddress(`fx1fba9c7f74010fe4068e476bbe9380fbd18d80d5`):{Balance:new(big.Int).SetStrings(`1211177999999999934464`,10)},
			common.HexToAddress(`fx1fe23cd51c750e84a25d32c3cb9e93c6eb7c0415`):{Balance:new(big.Int).SetStrings(`1374395000000000000`,10)},
			common.HexToAddress(`fx204db4b90964f1f0b07f4e84da63144b6ffc3b2d`):{Balance:new(big.Int).SetStrings(`22159000000000000`,10)},
			common.HexToAddress(`fx20b42f902c32b0156f39987e6608252e352b3df7`):{Balance:new(big.Int).SetStrings(`53948000000000000`,10)},
			common.HexToAddress(`fx211798cede83e23655724f07bb8e5c5e34ab7ec9`):{Balance:new(big.Int).SetStrings(`2998779000000000000`,10)},
			common.HexToAddress(`fx21e56b69f9fd3df9a82cd721f31cdc17942bae1e`):{Balance:new(big.Int).SetStrings(`3537979000000000000`,10)},
			common.HexToAddress(`fx21ee4a6ba64f2aed396aafac0170738ea87eb90e`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fx22156efef28e9ae727d5751ffc231950b690d694`):{Balance:new(big.Int).SetStrings(`62877800000000000000`,10)},
			common.HexToAddress(`fx22fcdf36e036616d1934d3e0c39c95f17f04d6ac`):{Balance:new(big.Int).SetStrings(`54589000000000000`,10)},
			common.HexToAddress(`fx230501386ff61363c4331cfa1db735f043dbccd3`):{Balance:new(big.Int).SetStrings(`230279000000000000`,10)},
			common.HexToAddress(`fx230cf2573f1f1d155fa4eb06a8d815d6d3d9b412`):{Balance:new(big.Int).SetStrings(`87034573999999991808`,10)},
			common.HexToAddress(`fx2351786349e5e41c84a0c68585b8c3ccb8c3c7b1`):{Balance:new(big.Int).SetStrings(`13590000000000000000`,10)},
			common.HexToAddress(`fx2360b0bcdbc743675e624fc28314d1ce8e7eae9b`):{Balance:new(big.Int).SetStrings(`115405179000000004096`,10)},
			common.HexToAddress(`fx25013805364c6b8333dd87511c9eb90736addb5c`):{Balance:new(big.Int).SetStrings(`433018419999999393792`,10)},
			common.HexToAddress(`fx25972e27351b1301ced9a30b8f4fbc18bebac028`):{Balance:new(big.Int).SetStrings(`16274579000000000000`,10)},
			common.HexToAddress(`fx26bf10b329e48f76f82da1a1e2b84f13e8c68fad`):{Balance:new(big.Int).SetStrings(`25824200000000000000`,10)},
			common.HexToAddress(`fx26cb5e0b3cdd85d2ee452eaf9c52d46008e310b3`):{Balance:new(big.Int).SetStrings(`44574400000000000000`,10)},
			common.HexToAddress(`fx2710289fdfaa821466170a3f330ef81253101758`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx282005900eda5b883dd5b76e98fb16e73d74baf7`):{Balance:new(big.Int).SetStrings(`6844900000000000000`,10)},
			common.HexToAddress(`fx28b65065bf3b9881d1a31a1e2d4ff6cb7809c2a2`):{Balance:new(big.Int).SetStrings(`9968400000000000000`,10)},
			common.HexToAddress(`fx29d130eb550941db8763da2504f93a5abf22addd`):{Balance:new(big.Int).SetStrings(`19270000000000000000`,10)},
			common.HexToAddress(`fx2a16bb8a652c9b40ca90afada7fb287527214926`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx2b490dcf2e53546005a3096d2c95a02e663d7abf`):{Balance:new(big.Int).SetStrings(`2000000000000000000`,10)},
			common.HexToAddress(`fx2b8910ae067f04cbaa6cd7501c62fd6e2416b407`):{Balance:new(big.Int).SetStrings(`124940800000000000000`,10)},
			common.HexToAddress(`fx2c97d257c49b955322c6c21d6f922869992bbe36`):{Balance:new(big.Int).SetStrings(`24265800000000000000`,10)},
			common.HexToAddress(`fx2cc87ac14edf3aa6b67fd31eefff801666d6f91c`):{Balance:new(big.Int).SetStrings(`131254658000000008192`,10)},
			common.HexToAddress(`fx2d196ca887b27ef1a42a263d0e0940932b43ae3c`):{Balance:new(big.Int).SetStrings(`2753600000000000000`,10)},
			common.HexToAddress(`fx2ddcba191a41134f5514396c803e81196c9cc4c1`):{Balance:new(big.Int).SetStrings(`173355382000000008192`,10)},
			common.HexToAddress(`fx2e40c050e7ff773aaed6dec65624bf6c4552a521`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx2e4a4aa31f3e1cfb433b98e8408238ad4fb5bc61`):{Balance:new(big.Int).SetStrings(`21270000000000000000`,10)},
			common.HexToAddress(`fx2e5871fe1fc67bfe0879813831c2cdb482d1c698`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx2e85d511be4c856e0f7ccb647c5d721ba15f6f1e`):{Balance:new(big.Int).SetStrings(`13428500000000000000`,10)},
			common.HexToAddress(`fx2e8659aeef76d397d89532ea8765c680ad8acb18`):{Balance:new(big.Int).SetStrings(`5166600000000000000`,10)},
			common.HexToAddress(`fx2f0136b7233c41c564e568174a3ffb1b7c8b20b0`):{Balance:new(big.Int).SetStrings(`401831832999999963136`,10)},
			common.HexToAddress(`fx2f23af02c1453bf7548db80a7505f3c7b2cdfb2e`):{Balance:new(big.Int).SetStrings(`26711000000000000`,10)},
			common.HexToAddress(`fx2ff61220d8e29cf5cb8299f73800e4c0fea92f72`):{Balance:new(big.Int).SetStrings(`17274600000000000000`,10)},
			common.HexToAddress(`fx303cecad4618e28668d65b4a02dcefbababe5bf5`):{Balance:new(big.Int).SetStrings(`508531100000000016384`,10)},
			common.HexToAddress(`fx3095fba647e27c9f6c6047a18d7c803a74ecc1ed`):{Balance:new(big.Int).SetStrings(`40300000000000000`,10)},
			common.HexToAddress(`fx30d4cb14e05cf49803b16b3e45c88c035cec8deb`):{Balance:new(big.Int).SetStrings(`175106579999999983616`,10)},
			common.HexToAddress(`fx31d53e78c2a585f4461562bd21b13dc997f41481`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx31f76d67b12978a3ae8b75f4b532953271135667`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx31fb73a8df7c1331221ea475a794364cbbc9e392`):{Balance:new(big.Int).SetStrings(`18781260000000000000`,10)},
			common.HexToAddress(`fx334b08e3ca8836b6af600f8cd7063a862f1f7a7d`):{Balance:new(big.Int).SetStrings(`491246416000000000000`,10)},
			common.HexToAddress(`fx3410d323fa685dc5c4189cadb92633d3cf440f51`):{Balance:new(big.Int).SetStrings(`217672178999999987712`,10)},
			common.HexToAddress(`fx343011ec839b02d078ac16ef8c01f1fb8e669121`):{Balance:new(big.Int).SetStrings(`115378879000000004096`,10)},
			common.HexToAddress(`fx344f0ce585cd18503dec25905fbc6f44a4c8c517`):{Balance:new(big.Int).SetStrings(`37743000000000000`,10)},
			common.HexToAddress(`fx353ff086dd5f7829273e6779117c2b58ee6b9208`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fx35665d6c2f7e61ab4266b10961a57a73f8b88978`):{Balance:new(big.Int).SetStrings(`21998800000000000000`,10)},
			common.HexToAddress(`fx35f818c03222500a9c4d3baa6181af215c9ec4a1`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx36093e0a375d7dc7840a9502da4c511b65a68b96`):{Balance:new(big.Int).SetStrings(`4265779000000000000`,10)},
			common.HexToAddress(`fx362d2a04b5c9438777b9c27ea2dfbf464d40bb84`):{Balance:new(big.Int).SetStrings(`4265779000000000000`,10)},
			common.HexToAddress(`fx37d9bd813e06ae084932cd3661673ff7cc0bb228`):{Balance:new(big.Int).SetStrings(`40706800000000000000`,10)},
			common.HexToAddress(`fx37fed9abb3a54609f72472294bee49492680ba8b`):{Balance:new(big.Int).SetStrings(`28917479000000000000`,10)},
			common.HexToAddress(`fx39010dc0af4a20f5f28660ebc75c9f1f0b3ce0e2`):{Balance:new(big.Int).SetStrings(`1000000000000000000`,10)},
			common.HexToAddress(`fx3a56916b01f532a294261149f92138e025088087`):{Balance:new(big.Int).SetStrings(`14947900000000000000`,10)},
			common.HexToAddress(`fx3ae87a2172b00c4a98d26699c1b664fbebe4e5f3`):{Balance:new(big.Int).SetStrings(`1000000000000000000`,10)},
			common.HexToAddress(`fx3b85e88c4631b7e8763bcdee128a733154fb61ac`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fx3c3d6e777ce84b86ada3d9254073c180c9a816c7`):{Balance:new(big.Int).SetStrings(`117003379000000004096`,10)},
			common.HexToAddress(`fx3c81c1994c9a665027c132758e12e9dc4040f3dd`):{Balance:new(big.Int).SetStrings(`8148500000000000000`,10)},
			common.HexToAddress(`fx3ca067b8f67f0c1cbe63b6fc1606d0c258fa11d8`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx3cdd36d9e5f717c824ebd5eb762dc851a9a52e6b`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx3d672ac10c09e2c398a3ce82b19e5e04b7d04e38`):{Balance:new(big.Int).SetStrings(`524682052999999979520`,10)},
			common.HexToAddress(`fx3ebbc31f468ce12b038e5823541478dd5a0f5010`):{Balance:new(big.Int).SetStrings(`5480400000000000000`,10)},
			common.HexToAddress(`fx3f5adbfa463bb7001899d24f400fea2adb975699`):{Balance:new(big.Int).SetStrings(`24265779000000000000`,10)},
			common.HexToAddress(`fx4061ee9628618977ec7a650b2302ad1b0d524970`):{Balance:new(big.Int).SetStrings(`209119479000000004096`,10)},
			common.HexToAddress(`fx416d723fae36fc9b650386fef07047e6fdf895fe`):{Balance:new(big.Int).SetStrings(`27000000000000`,10)},
			common.HexToAddress(`fx419b531e29f8e31d24ff461ce14a0b5513ca0a90`):{Balance:new(big.Int).SetStrings(`7235716000000000000`,10)},
			common.HexToAddress(`fx41a0c63204fbd95011a1d5f4cea3d365c8764991`):{Balance:new(big.Int).SetStrings(`5550000000000000`,10)},
			common.HexToAddress(`fx41ceeb1f0bf4ca7db7b109ae4708da42b2b87b28`):{Balance:new(big.Int).SetStrings(`14423000000000000000`,10)},
			common.HexToAddress(`fx41e12c18f1de984c432fd06aa78ba42726134739`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fx421a849b63273af4505d3ce6a1b46b0849028e2d`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx435a7ef976fe3b979313f2ada19e582e42b5ed7c`):{Balance:new(big.Int).SetStrings(`13631100000000000000`,10)},
			common.HexToAddress(`fx4393725e3b12b56e372881afa4ce13a1d12d4329`):{Balance:new(big.Int).SetStrings(`170056775029999992832`,10)},
			common.HexToAddress(`fx44e27280d6e9d4ac6a2da7b0d0daa6d841ef1a43`):{Balance:new(big.Int).SetStrings(`105658679000000004096`,10)},
			common.HexToAddress(`fx454e87142b8cdea3949d4c0cfc5860f8ce000429`):{Balance:new(big.Int).SetStrings(`264811756999999979520`,10)},
			common.HexToAddress(`fx4552745c798b90b32812d1094fd2b8c687a5e31a`):{Balance:new(big.Int).SetStrings(`4211800000000000000`,10)},
			common.HexToAddress(`fx45b08c870790ed0d63565261a2490d8b3d6a9e04`):{Balance:new(big.Int).SetStrings(`3487000000000000000`,10)},
			common.HexToAddress(`fx45ca5f78d73670ea2ad32fe43e1e4505d78c8367`):{Balance:new(big.Int).SetStrings(`1052503000000000000`,10)},
			common.HexToAddress(`fx469454c8689514c96c7bb8db81370a2b87c29c41`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fx46cdb7921384e60fab685587351c2b33eae2c637`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx483cd7d754c398c06434300f4de38420e9cf3c1a`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx48ab0da91f113076e157b5bc2e44a50bd1296c21`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx49cfcac5a91bae83562f92513434e55d57909c02`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fx4a35c57c672650d2a492416ed1186e2333fb9dcb`):{Balance:new(big.Int).SetStrings(`11152769000000000000`,10)},
			common.HexToAddress(`fx4ac2112213c109db40691eb6ba4916c7902465e8`):{Balance:new(big.Int).SetStrings(`610859115999999950848`,10)},
			common.HexToAddress(`fx4bcf9ab48ea39f2f8720a1adc1b04625bce7abcc`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx4c0ff62b07918f535f632192dc662833b9b55967`):{Balance:new(big.Int).SetStrings(`2985160000000000000`,10)},
			common.HexToAddress(`fx4ea2b6ae3ac6d25b596f483d5a2554cf97746116`):{Balance:new(big.Int).SetStrings(`3328800000000000000`,10)},
			common.HexToAddress(`fx4ff6aa0985ff4f3757eb273e5c2e988d5e62eebd`):{Balance:new(big.Int).SetStrings(`2794833000000000000`,10)},
			common.HexToAddress(`fx503f8cc13a3ea056d75822b44332314da1fba034`):{Balance:new(big.Int).SetStrings(`9505000000000000000`,10)},
			common.HexToAddress(`fx5040673682f1a42152ad6ec27f7be93d340a1411`):{Balance:new(big.Int).SetStrings(`10870600000000000000`,10)},
			common.HexToAddress(`fx50de39629fc8e68b22d0e182b475714deb8f8c0d`):{Balance:new(big.Int).SetStrings(`37544600000000000000`,10)},
			common.HexToAddress(`fx516dd07972ba8e8d13fe9701fe0a2ea8b8f5b100`):{Balance:new(big.Int).SetStrings(`6957300000000000000`,10)},
			common.HexToAddress(`fx5225a647e748280d7da1402f89c0230c1c41783e`):{Balance:new(big.Int).SetStrings(`175210363469999996928`,10)},
			common.HexToAddress(`fx52f34d2036692956903a298a60a4296076571898`):{Balance:new(big.Int).SetStrings(`6045600000000000000`,10)},
			common.HexToAddress(`fx53440d811c1221f5995d627de0e54a72c8e07292`):{Balance:new(big.Int).SetStrings(`34535000000000000000`,10)},
			common.HexToAddress(`fx535eeb0ee758c1575362d13fb549e9074f9b69b2`):{Balance:new(big.Int).SetStrings(`4092600000000000000`,10)},
			common.HexToAddress(`fx53670bf8db176acb524d90b1df18a3b733543957`):{Balance:new(big.Int).SetStrings(`87001152999999995904`,10)},
			common.HexToAddress(`fx54af5abfe98759aafe599c9cb20378db8cbc65ea`):{Balance:new(big.Int).SetStrings(`2998779000000000000`,10)},
			common.HexToAddress(`fx54bbd5bed287180fcb0f2daa6f7d8488a2a47b01`):{Balance:new(big.Int).SetStrings(`999979000000000000`,10)},
			common.HexToAddress(`fx555f8fefaf6c8df1d2f23a20b4a18263eb372f5e`):{Balance:new(big.Int).SetStrings(`181402200000000000000`,10)},
			common.HexToAddress(`fx557e551893dacc032c09dce0b949bff9fadc7320`):{Balance:new(big.Int).SetStrings(`6798500000000000000`,10)},
			common.HexToAddress(`fx5697e6b87f62564a0287e803e344112047d5591f`):{Balance:new(big.Int).SetStrings(`7554352999999819776`,10)},
			common.HexToAddress(`fx56a56908585622ee48268085a6de9ef9e564737d`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx56e80db9e8d5a6cbc9ac2b74a6e7e89b0632876b`):{Balance:new(big.Int).SetStrings(`37544600000000000000`,10)},
			common.HexToAddress(`fx58235a9f53769d8a1c4e8c9a5a1c0ca99f95fbf1`):{Balance:new(big.Int).SetStrings(`20377100000000000000`,10)},
			common.HexToAddress(`fx584b8d21cdb0b47128162e68053227a988058b84`):{Balance:new(big.Int).SetStrings(`26150794000000000000`,10)},
			common.HexToAddress(`fx5893c6afef98a2543919c084a6f7999ab269b1f6`):{Balance:new(big.Int).SetStrings(`14423000000000000000`,10)},
			common.HexToAddress(`fx58eb2c9ba1c9a967f21b03a4fa9270f58cd72e46`):{Balance:new(big.Int).SetStrings(`33946200000000000000`,10)},
			common.HexToAddress(`fx5b10e5c6b2c3add0d54441ab2bcac61b4b0fd550`):{Balance:new(big.Int).SetStrings(`3192153000000000000`,10)},
			common.HexToAddress(`fx5c04879dce302e2d2e8bf1e8c658ffbec1a63df6`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx5c27b260fcc84b23871852ac1b72add3335ea677`):{Balance:new(big.Int).SetStrings(`17465660000000000000`,10)},
			common.HexToAddress(`fx5c3b6fa8d4aa103b4e46933116c5348bbe0fbf84`):{Balance:new(big.Int).SetStrings(`7864099000000000000`,10)},
			common.HexToAddress(`fx5cb9c3c2f0638b306c66ff0047e630b8bc4d3135`):{Balance:new(big.Int).SetStrings(`3265769000000000000`,10)},
			common.HexToAddress(`fx5cc098759dc30de6422e3836f709a1e2dd6ff523`):{Balance:new(big.Int).SetStrings(`31970000000000000000`,10)},
			common.HexToAddress(`fx5d24c8d9a27778b67d2e4a25b2f3da4cd5162358`):{Balance:new(big.Int).SetStrings(`4971279000000000000`,10)},
			common.HexToAddress(`fx5d2abd64c269199b6fb6ea3196bcfb49b562016c`):{Balance:new(big.Int).SetStrings(`7201806000000000000`,10)},
			common.HexToAddress(`fx5d3138a55f95e66e3baf34d5045edd13096b916e`):{Balance:new(big.Int).SetStrings(`16341811000000000000`,10)},
			common.HexToAddress(`fx5f36cfb49c5957269481fd4afc165b4ecb70f1b2`):{Balance:new(big.Int).SetStrings(`11270337000000000000`,10)},
			common.HexToAddress(`fx606937f8df63622571711c52a4434e31d2691c4c`):{Balance:new(big.Int).SetStrings(`1077876978999999987712`,10)},
			common.HexToAddress(`fx609a03874e577504b635030fedb36f24cca92d2f`):{Balance:new(big.Int).SetStrings(`2620100000000000000`,10)},
			common.HexToAddress(`fx60b15b84359fd54703f6b6b0495723141306ffe5`):{Balance:new(big.Int).SetStrings(`26977000000000000`,10)},
			common.HexToAddress(`fx60ee924a8b73f967ca631670fcf5c80540ebfcfe`):{Balance:new(big.Int).SetStrings(`999979000000000000`,10)},
			common.HexToAddress(`fx61687695dd6a6669b9868417017f644473ca99ce`):{Balance:new(big.Int).SetStrings(`17561785000000000000`,10)},
			common.HexToAddress(`fx62372fb809222de866019186e96c7cf39234a157`):{Balance:new(big.Int).SetStrings(`103970878800000024576`,10)},
			common.HexToAddress(`fx6385e7e2426a211fa0723126e6d86273f6bfe019`):{Balance:new(big.Int).SetStrings(`62229000000000000`,10)},
			common.HexToAddress(`fx638bb40ab936f719644f59415f0de69e2b10aa15`):{Balance:new(big.Int).SetStrings(`769992000000000000000`,10)},
			common.HexToAddress(`fx643f31eb9b83869513526521436d728e11e176b2`):{Balance:new(big.Int).SetStrings(`25579800000000000000`,10)},
			common.HexToAddress(`fx6447b9850876a826059e14f4468085694d341acb`):{Balance:new(big.Int).SetStrings(`413931182000000008192`,10)},
			common.HexToAddress(`fx6560927aff5a6fbda572a4182f638ce8d32a482f`):{Balance:new(big.Int).SetStrings(`19230253000000000000`,10)},
			common.HexToAddress(`fx656b08ab21bec7d5bfb93ad89f057cecd22d8861`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fx65a7b52456d32e5be874e09701945ae72c6e86fa`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fx65e25f1aae37404f9107a3fe5b024f3580cf1ec3`):{Balance:new(big.Int).SetStrings(`4265779000000000000`,10)},
			common.HexToAddress(`fx672afc7a4f0a81222d22cee256af882b7359172b`):{Balance:new(big.Int).SetStrings(`2620314947999999983616`,10)},
			common.HexToAddress(`fx680fbc5582c6acf9144e0ca18ebc5fc19a6ffa02`):{Balance:new(big.Int).SetStrings(`1285993000000000000`,10)},
			common.HexToAddress(`fx682f72de51149859b7a04d38b3fa4f1a3ad8e862`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fx6885e558c8ce77aeaedd1200faa4b534d2dd08fc`):{Balance:new(big.Int).SetStrings(`3379553000000000000`,10)},
			common.HexToAddress(`fx6887b82225b763d5d567da151baf4383ed35073b`):{Balance:new(big.Int).SetStrings(`501882389999999975424`,10)},
			common.HexToAddress(`fx68e79d7a19f233138329db3fc408678a5f981148`):{Balance:new(big.Int).SetStrings(`809960997200000057344`,10)},
			common.HexToAddress(`fx690885bccb82f63e49ed643748ab62a741a734d8`):{Balance:new(big.Int).SetStrings(`28846000000000000000`,10)},
			common.HexToAddress(`fx6979664c8a71f26394db49957b22fbc38036b096`):{Balance:new(big.Int).SetStrings(`20377100000000000000`,10)},
			common.HexToAddress(`fx697b84e5363bb3541925546d71048143797f7b51`):{Balance:new(big.Int).SetStrings(`4771216000000020480`,10)},
			common.HexToAddress(`fx6a7f91367852fef023b3d476dd3a8b803a449c45`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx6b8757d9665c81eb8a741200b7bdcf4b61c0e1ae`):{Balance:new(big.Int).SetStrings(`202612158000000008192`,10)},
			common.HexToAddress(`fx6bbacf2ca0a80ca782804291699dc932079ffba0`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fx6c1584d7af56059bf74f418312c5fa0ba1782475`):{Balance:new(big.Int).SetStrings(`411213648000000000000`,10)},
			common.HexToAddress(`fx6cf85cbaa63659d66729114a5fe25cee59788f04`):{Balance:new(big.Int).SetStrings(`9716359000000000000`,10)},
			common.HexToAddress(`fx6d6023386c741e48b05922d9a66c9363aa4f7f95`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx6e11f763bd9e9abd20559342bf20fa1cc4e476b9`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx6f6368d39d0039857882437844456f6a7f5ad9f7`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fx6f879a5ba220e133876e0e1b854e32a6eda3c803`):{Balance:new(big.Int).SetStrings(`99877623999999995958788096`,10)},
			common.HexToAddress(`fx6fbaaf7231f697e49b30c1eea0db8c3d0a042749`):{Balance:new(big.Int).SetStrings(`22844000000000000000`,10)},
			common.HexToAddress(`fx6ff0893328eb52872905f14bd563626202969d9f`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx7020d4672a2bf67eb9d0ea49b430d9bba07bddc1`):{Balance:new(big.Int).SetStrings(`407080649469999972352`,10)},
			common.HexToAddress(`fx71244fbfaa1d253b98b2bee1fb511ef697d5d462`):{Balance:new(big.Int).SetStrings(`4265779000000000000`,10)},
			common.HexToAddress(`fx71d251ff05ea350ff1122de60607576d8d0f764b`):{Balance:new(big.Int).SetStrings(`66595900000000000000`,10)},
			common.HexToAddress(`fx725dcfee2cf7be187d122c4b8d6164bdfb91efa4`):{Balance:new(big.Int).SetStrings(`444700000000000000`,10)},
			common.HexToAddress(`fx72a845e353cfd45189306dc8d9839134349ed070`):{Balance:new(big.Int).SetStrings(`14330579000000000000`,10)},
			common.HexToAddress(`fx73128f422ee5fab47e1e0a7ed30811d88edcc735`):{Balance:new(big.Int).SetStrings(`5765779000000000000`,10)},
			common.HexToAddress(`fx7346c7a719a9feb902d1549a7e08a99fd97ec34f`):{Balance:new(big.Int).SetStrings(`437810936999999963136`,10)},
			common.HexToAddress(`fx739183269a40998b1c8040936fda3db4309a5e92`):{Balance:new(big.Int).SetStrings(`41235800000000000000`,10)},
			common.HexToAddress(`fx75578d65c215d3cbe97ca8271653d2c2508411a4`):{Balance:new(big.Int).SetStrings(`2480379000000000000`,10)},
			common.HexToAddress(`fx761bd8ff275de3e952074837a5221e83c8a5725b`):{Balance:new(big.Int).SetStrings(`217275499999999983616`,10)},
			common.HexToAddress(`fx76274e04dad753a2e6d9ccfcb45253d091347ea7`):{Balance:new(big.Int).SetStrings(`3389179000000000000`,10)},
			common.HexToAddress(`fx764d0e670d3222f672fe08bb5ee96a3f5fd05027`):{Balance:new(big.Int).SetStrings(`3487000000000000000`,10)},
			common.HexToAddress(`fx777a9b21070cb6b7121210830df8972799d012cf`):{Balance:new(big.Int).SetStrings(`243072578999999987712`,10)},
			common.HexToAddress(`fx786af7a8bc9ba06eeef82df604da2b5281271cb3`):{Balance:new(big.Int).SetStrings(`87124873999999991808`,10)},
			common.HexToAddress(`fx787b410171e7f19d34f5e8066d9a1dbf5979c54b`):{Balance:new(big.Int).SetStrings(`1004938906999999954944`,10)},
			common.HexToAddress(`fx788db1af36f3c4a0f01e2657d9bdeb7f0b47379d`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx791f27a516396d923b2ee722b2bff5b39cb0533a`):{Balance:new(big.Int).SetStrings(`24290300000000000000`,10)},
			common.HexToAddress(`fx792553fcf99000ac19328ad148ee7518226958cb`):{Balance:new(big.Int).SetStrings(`38863270000000008192`,10)},
			common.HexToAddress(`fx7926fdca0b62570522fdf433fcfd1b429e6b48a8`):{Balance:new(big.Int).SetStrings(`43905800000000000000`,10)},
			common.HexToAddress(`fx79a053e676c588c4151d814a3b0fd4298bbe012d`):{Balance:new(big.Int).SetStrings(`3472216000000000000`,10)},
			common.HexToAddress(`fx79ba63a6b215cdcb17911fc2bda4bed8b6ee6de5`):{Balance:new(big.Int).SetStrings(`6790632000000000000`,10)},
			common.HexToAddress(`fx79e884c0eca3615c62a30b7956b9e0352c8d051a`):{Balance:new(big.Int).SetStrings(`427445798999999971328`,10)},
			common.HexToAddress(`fx7a24f0adb9a373e73a9d4cf63edd840dd113c166`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx7a3b59ba4eb1792b0e77b8f737c20e9bc0b61d45`):{Balance:new(big.Int).SetStrings(`516856776000000032768`,10)},
			common.HexToAddress(`fx7a624930e426ad8c3137ef604121662136d2811a`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx7bb4b4ef55cb6c3d080adcc67c9004e812c7df52`):{Balance:new(big.Int).SetStrings(`2734600000000000000`,10)},
			common.HexToAddress(`fx7c92ed48a002a7292a794bc352001d7db727b6be`):{Balance:new(big.Int).SetStrings(`7508095000000000000`,10)},
			common.HexToAddress(`fx7d66032650fa914dd70e9daff07287efef8d0636`):{Balance:new(big.Int).SetStrings(`11556600000000000000`,10)},
			common.HexToAddress(`fx7dd2ec9e1dde2ce8a9092053d88d2915eaf73e77`):{Balance:new(big.Int).SetStrings(`5480400000000000000`,10)},
			common.HexToAddress(`fx7e6a9dd329eedf8615602bee518475b4b2609a77`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx7f7deddd9af3012047ed7b07d2cd5f183b69fd09`):{Balance:new(big.Int).SetStrings(`504603037000000012288`,10)},
			common.HexToAddress(`fx7f88bb5a7a1492981830313427026fa6e629f5eb`):{Balance:new(big.Int).SetStrings(`232392147999999983616`,10)},
			common.HexToAddress(`fx7f9239ec4431f9c66b6cc8d2430ff1aaada35395`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fx8136b907c36c0212183bc9ef3a0794563baa12a6`):{Balance:new(big.Int).SetStrings(`38540000000000000000`,10)},
			common.HexToAddress(`fx81ab0dbb7303663fd2e94c7290e0b5475a8b6513`):{Balance:new(big.Int).SetStrings(`99248100000000000000`,10)},
			common.HexToAddress(`fx81bd18b8cfae27edf7fae7601e1d397d63c6a35e`):{Balance:new(big.Int).SetStrings(`212450880000000000000`,10)},
			common.HexToAddress(`fx82a1b18e567f36767e02df3c4a7e815ac2fd32ea`):{Balance:new(big.Int).SetStrings(`3653404000000000000`,10)},
			common.HexToAddress(`fx830e649757d0a716e3bf9d1d389811114c03f450`):{Balance:new(big.Int).SetStrings(`2828073999999991808`,10)},
			common.HexToAddress(`fx83b91bf498e7f45656eeedb4fecf3d3da176b63f`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx841202d77ab9e180301b18191c98f8ea31683715`):{Balance:new(big.Int).SetStrings(`5284392000000000000`,10)},
			common.HexToAddress(`fx8421a6cc88f3910d54603163295cef05dce34eec`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx8564aba1f7e51005e81a264afe2b5cff537f0372`):{Balance:new(big.Int).SetStrings(`125295000000000000`,10)},
			common.HexToAddress(`fx8573f279c3408949ddba1817e2a4765b4c693032`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fx8659c8d92896f87a92a808ffe6766b1a3695b847`):{Balance:new(big.Int).SetStrings(`5498118000000000000`,10)},
			common.HexToAddress(`fx86fadeddfd5e7d101be703ca3b6980a990250a26`):{Balance:new(big.Int).SetStrings(`10870600000000000000`,10)},
			common.HexToAddress(`fx870a1e3b090493fc33f0097b1e1b5f4df5108ab5`):{Balance:new(big.Int).SetStrings(`18842458000000000000`,10)},
			common.HexToAddress(`fx8855486a0f79b2e4f71e7d9cb7ada892ce06c28d`):{Balance:new(big.Int).SetStrings(`524881616000000000000`,10)},
			common.HexToAddress(`fx8a8e90c8d1b9375301016bd690c090776cb90672`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx8a96201969eed16de6afedf2985ebcdc8e1a517f`):{Balance:new(big.Int).SetStrings(`36128116000000000000`,10)},
			common.HexToAddress(`fx8acb044ca4b6bc1d8948ac4ee18e81fea8455bcd`):{Balance:new(big.Int).SetStrings(`7897974900000000311296`,10)},
			common.HexToAddress(`fx8ad7d4301caf8243774669ca31e19e6cf180b0c8`):{Balance:new(big.Int).SetStrings(`17665400000000000000`,10)},
			common.HexToAddress(`fx8b36e3ed2feb74109bf39f486cc85511a43c4b34`):{Balance:new(big.Int).SetStrings(`61214378999999987712`,10)},
			common.HexToAddress(`fx8be6e29e08418a515c05679e234bc29f790887ac`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fx8bf09ddf1af19d01d9e2ef11b2ae3bc458f04528`):{Balance:new(big.Int).SetStrings(`22307537000000000000`,10)},
			common.HexToAddress(`fx8d0cfe408997b0c79d1b0999c9825407029437ba`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx8d187d65b9fdbcce928fd7eba85c6e0913393205`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fx8d5893cc406e599f36650d3c0e24752c3fd529d9`):{Balance:new(big.Int).SetStrings(`1227275678999999938560`,10)},
			common.HexToAddress(`fx8f13477925260f54da4712e0e06501a836b6b4aa`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx8f32ff4704a013f9c05949c8a52c38f4db91858e`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx8faa12e5e4bf6de730596c2c6167fc1eef466da5`):{Balance:new(big.Int).SetStrings(`833493379000000053248`,10)},
			common.HexToAddress(`fx8fd1fbd534ff11c1bba8dee0e1f148aec8227464`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fx901c43af0a8271fd06bfe790d50210532fe7722a`):{Balance:new(big.Int).SetStrings(`1232748000000000000`,10)},
			common.HexToAddress(`fx90439186cacc244a33b25f8fcabaf5a20990a590`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fx90b18ed61f45c26df5633ee1358e0b8119e00fad`):{Balance:new(big.Int).SetStrings(`6208879189999999778816`,10)},
			common.HexToAddress(`fx91cc7848553e2a4bc367fb7ee9a7177e866e69ea`):{Balance:new(big.Int).SetStrings(`6045600000000000000`,10)},
			common.HexToAddress(`fx91eeee151bbb39d1de160a3c580cf339516bc24b`):{Balance:new(big.Int).SetStrings(`3266811000000000000`,10)},
			common.HexToAddress(`fx9255071b81fbcf4b7b53d9eea04d36f44e477432`):{Balance:new(big.Int).SetStrings(`59903896999999995904`,10)},
			common.HexToAddress(`fx934668670795a4768ac8ff9ab04c9f93fa1ba7dd`):{Balance:new(big.Int).SetStrings(`9111300000000000000`,10)},
			common.HexToAddress(`fx947dd3d1af0a0b502f42d17034ef2897ba0aec77`):{Balance:new(big.Int).SetStrings(`164527000000000000`,10)},
			common.HexToAddress(`fx9500e05bfb0df8cb1e798f87798432b7f6d0fc98`):{Balance:new(big.Int).SetStrings(`1511300000000000000`,10)},
			common.HexToAddress(`fx9510caa3aa0c5c1b4ceeca75659a3f95c85bafe1`):{Balance:new(big.Int).SetStrings(`24265800000000000000`,10)},
			common.HexToAddress(`fx9511057fed70c87d136c82dc373eda3769fcab50`):{Balance:new(big.Int).SetStrings(`24265800000000000000`,10)},
			common.HexToAddress(`fx967d34f16ddcfe3a5924e1686e1fb1a50c3f79db`):{Balance:new(big.Int).SetStrings(`13692600000000000000`,10)},
			common.HexToAddress(`fx96f3d59b5876a63498d957faf74f16d8107c9dbc`):{Balance:new(big.Int).SetStrings(`41325379000000004096`,10)},
			common.HexToAddress(`fx9825ad9e579f1a174afa59db51678d84a60e8ce4`):{Balance:new(big.Int).SetStrings(`62379000000000000`,10)},
			common.HexToAddress(`fx997807b7d4ab9035f81f8f5f007ec093e7d041a3`):{Balance:new(big.Int).SetStrings(`3318700000000000000`,10)},
			common.HexToAddress(`fx99ae34fe1a3edbc6dfd6c64ec404f208e445b902`):{Balance:new(big.Int).SetStrings(`1842859000000000000`,10)},
			common.HexToAddress(`fx9a8d5762214f3b5c84e9e47384123c102faf4b53`):{Balance:new(big.Int).SetStrings(`1008517569000000061440`,10)},
			common.HexToAddress(`fx9b7d63a84814737fb7a29e094ad814ccee01e220`):{Balance:new(big.Int).SetStrings(`7780158000000000000`,10)},
			common.HexToAddress(`fx9babcae0b65a7e9e06ad2c836ffd411afe81465a`):{Balance:new(big.Int).SetStrings(`28472561142930000052224`,10)},
			common.HexToAddress(`fx9c6bbde319e6ac7075d92eb7f6b41881430939c0`):{Balance:new(big.Int).SetStrings(`2722080000000000000`,10)},
			common.HexToAddress(`fx9db6fc0e46c7396232570e98f77ad0eb32d6ab77`):{Balance:new(big.Int).SetStrings(`16160700000000000000`,10)},
			common.HexToAddress(`fx9df6f6b6913d47b12a8a376a6deb4eacfce5a77b`):{Balance:new(big.Int).SetStrings(`805137573999999975424`,10)},
			common.HexToAddress(`fx9e2a1bbe4240fc6a0751030cfa359cb314d6d4ae`):{Balance:new(big.Int).SetStrings(`54106100000000000000`,10)},
			common.HexToAddress(`fx9e67603585eea82b874e1e727bbc6412362209d4`):{Balance:new(big.Int).SetStrings(`12227700000000000000`,10)},
			common.HexToAddress(`fx9ed1fbdd32f2d221240341ffab560998f0f31efb`):{Balance:new(big.Int).SetStrings(`83988500000000000000`,10)},
			common.HexToAddress(`fxa012d54dfef2ae2fc230fbc926c164a05c470225`):{Balance:new(big.Int).SetStrings(`2679552726999999971328`,10)},
			common.HexToAddress(`fxa125e6ad1eba8a8546c85d14c10e79776bb148e2`):{Balance:new(big.Int).SetStrings(`258506174000000008192`,10)},
			common.HexToAddress(`fxa17c52d69b3eaa2705d58a2dde0fc17cec442886`):{Balance:new(big.Int).SetStrings(`29896454000000000000`,10)},
			common.HexToAddress(`fxa1f3e4e1d4416d10bc5d9b101c946c47657a156c`):{Balance:new(big.Int).SetStrings(`927937000000000000`,10)},
			common.HexToAddress(`fxa2264c3190a54a661b055b44474b95902ca6e1f6`):{Balance:new(big.Int).SetStrings(`19916200000000000000`,10)},
			common.HexToAddress(`fxa39b320d42dc925d13b1d65fa09923acae5c8753`):{Balance:new(big.Int).SetStrings(`6844900000000000000`,10)},
			common.HexToAddress(`fxa3fd312e152f65858567920b34b49e937211771e`):{Balance:new(big.Int).SetStrings(`3990812000000000000`,10)},
			common.HexToAddress(`fxa4fc98b1ae44e08aefa0c06fbc1009e2f034c87e`):{Balance:new(big.Int).SetStrings(`43905800000000000000`,10)},
			common.HexToAddress(`fxa532a8c90ade9bf59865b04d486b51c0ba41d379`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fxa57f950d2fd8b532b9987032d0a96e99a0255e6d`):{Balance:new(big.Int).SetStrings(`6267130000000000000`,10)},
			common.HexToAddress(`fxa5c73d5362d227ea97c9ce102954df3091f01be7`):{Balance:new(big.Int).SetStrings(`5374479000000000000`,10)},
			common.HexToAddress(`fxa76d6651e31fdbd032ffea0423e072406ffb45f3`):{Balance:new(big.Int).SetStrings(`170821555999999983616`,10)},
			common.HexToAddress(`fxa78aad340dfe2ed51f26ba8fd45f9b903d3a63bd`):{Balance:new(big.Int).SetStrings(`255856600000000000000`,10)},
			common.HexToAddress(`fxa7bbb4474bc7fa76328377b49190b6f841eb4c72`):{Balance:new(big.Int).SetStrings(`999979000000000000`,10)},
			common.HexToAddress(`fxa82b99ddf95a9786cd9924b30a5810e24f0eeb96`):{Balance:new(big.Int).SetStrings(`513782234000000024576`,10)},
			common.HexToAddress(`fxa8beade0845c2dff89ec3928fc8458cd54b740ee`):{Balance:new(big.Int).SetStrings(`6844900000000000000`,10)},
			common.HexToAddress(`fxa9114957e2558e0e8a50b7080871eda15dee8a7c`):{Balance:new(big.Int).SetStrings(`7429016099000099840`,10)},
			common.HexToAddress(`fxaa29788dbe3431d92ad80a89769a1b48b771cf23`):{Balance:new(big.Int).SetStrings(`499958000000000000`,10)},
			common.HexToAddress(`fxaa772639f4d03d56095fa17ccda574de63991cba`):{Balance:new(big.Int).SetStrings(`624770415999999934464`,10)},
			common.HexToAddress(`fxaac1cd5c077e9364c0c8a46843b7a43749e9e435`):{Balance:new(big.Int).SetStrings(`40508954000000000000`,10)},
			common.HexToAddress(`fxaadd9ccfbc21c201369455e52bbeec894c392071`):{Balance:new(big.Int).SetStrings(`57374500000000000000`,10)},
			common.HexToAddress(`fxab271af28b768a47993e01c460dc5e9a6b24660e`):{Balance:new(big.Int).SetStrings(`37163000000000000000`,10)},
			common.HexToAddress(`fxab4ea140b939884f7e0be211b662828363a43f9a`):{Balance:new(big.Int).SetStrings(`204784478000000008192`,10)},
			common.HexToAddress(`fxab58b678c30b05c01b0d97fe6059b4264cc224c2`):{Balance:new(big.Int).SetStrings(`15460358000000000000`,10)},
			common.HexToAddress(`fxabddcf3c84f56379ab8b3d01e733413d6004fa78`):{Balance:new(big.Int).SetStrings(`25265800000000000000`,10)},
			common.HexToAddress(`fxac0e29b1fa67f903365ec5fb9d974cfcb292660a`):{Balance:new(big.Int).SetStrings(`26690000000000000`,10)},
			common.HexToAddress(`fxaca597b464551d66bdce88ff051c7297ed527fb1`):{Balance:new(big.Int).SetStrings(`2719800000000000000`,10)},
			common.HexToAddress(`fxad3465da2cbfb34c16d6b4301ab209de923f3880`):{Balance:new(big.Int).SetStrings(`265534997000000012288`,10)},
			common.HexToAddress(`fxae1dd56103296cd1d9783ad6b199d992ff17a808`):{Balance:new(big.Int).SetStrings(`1627247999999991808`,10)},
			common.HexToAddress(`fxae98aa016e9822bdd5cee1ff720b8651d9a91583`):{Balance:new(big.Int).SetStrings(`1541611885000000012288`,10)},
			common.HexToAddress(`fxaee41944629b65bd0b875bb3c9e09cb7012c27e4`):{Balance:new(big.Int).SetStrings(`20534800000000000000`,10)},
			common.HexToAddress(`fxaf70392b2407c004f86b58de28323190b74b2067`):{Balance:new(big.Int).SetStrings(`54374458000000000000`,10)},
			common.HexToAddress(`fxafc242726c084c762c3d5d541cb5cbce7544bed6`):{Balance:new(big.Int).SetStrings(`9505000000000000000`,10)},
			common.HexToAddress(`fxaff33ffef696eb5dc45995662c95034c9c563959`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fxb0534b7ff81ad58b292448df081744a6fdcb8fd7`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fxb0e3e512abe6740b426dc7c900c1e44915a47112`):{Balance:new(big.Int).SetStrings(`208800974000000008192`,10)},
			common.HexToAddress(`fxb1075fdb8a38aef322e1c16cc1350227ca85ee41`):{Balance:new(big.Int).SetStrings(`7499737000000000000`,10)},
			common.HexToAddress(`fxb170d9be9c3f0bf659a2dcf709a15b54f8df2de0`):{Balance:new(big.Int).SetStrings(`36467900000000000000`,10)},
			common.HexToAddress(`fxb30a294bc149c2ddd8c51418932e511b58f1170c`):{Balance:new(big.Int).SetStrings(`21374479000000000000`,10)},
			common.HexToAddress(`fxb32fbf35f1b13badb5c74b455bb1e49b38d19591`):{Balance:new(big.Int).SetStrings(`41235800000000000000`,10)},
			common.HexToAddress(`fxb45059c5d818124255b5e3a77e5d55730c5f721b`):{Balance:new(big.Int).SetStrings(`14423000000000000000`,10)},
			common.HexToAddress(`fxb59039e4a8c380ef02226be019ff64645284b94c`):{Balance:new(big.Int).SetStrings(`14171418000000000000`,10)},
			common.HexToAddress(`fxb5f86c0f35d3fa8748dc09cea6aa744524d58c35`):{Balance:new(big.Int).SetStrings(`8322559000000000000`,10)},
			common.HexToAddress(`fxb6a45298f8809bd6d0fb5d888e7299d0e14646ec`):{Balance:new(big.Int).SetStrings(`2732100000000000000`,10)},
			common.HexToAddress(`fxb6e7be03c1ea3ed40fc0a39c23a36143c4136166`):{Balance:new(big.Int).SetStrings(`345745053000000012288`,10)},
			common.HexToAddress(`fxb77ee6ffae27b5d1d8984e5248e21cffa9ad357c`):{Balance:new(big.Int).SetStrings(`24265800000000000000`,10)},
			common.HexToAddress(`fxb7831aa90f6318151fad63dd74e9f4d86a24f55b`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fxb7960812d3371f95680cc1cd153150cae3f38ce7`):{Balance:new(big.Int).SetStrings(`2131011000000000000`,10)},
			common.HexToAddress(`fxb89ee53937286a02e28fbdc74bc0904d35855ffd`):{Balance:new(big.Int).SetStrings(`16309300000000000000`,10)},
			common.HexToAddress(`fxb939c546e418d7f87e4e1fe9779ede909241026d`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fxb95e641ce881b736c85c853d6ac767b7451e9323`):{Balance:new(big.Int).SetStrings(`1500000000000000000000`,10)},
			common.HexToAddress(`fxbaa07121defabf57497e2137eef1e6699f6d4efe`):{Balance:new(big.Int).SetStrings(`67092300000000000000`,10)},
			common.HexToAddress(`fxbac789f1a5c75daac76df7579241a0b7df7ea794`):{Balance:new(big.Int).SetStrings(`117592395000000004096`,10)},
			common.HexToAddress(`fxbb138a212892fc56995aa92e83685bddbf07eb1b`):{Balance:new(big.Int).SetStrings(`647879000000000000`,10)},
			common.HexToAddress(`fxbb34256f0e2a1b233429e6903a063d15c63c1bd5`):{Balance:new(big.Int).SetStrings(`117325000000000000000`,10)},
			common.HexToAddress(`fxbb4c91e0c021862fb9c52ab5843c5973595d2075`):{Balance:new(big.Int).SetStrings(`1037501000000000000`,10)},
			common.HexToAddress(`fxbb9f67bd5a18427dc769df7f3955e58f5a5ab4e3`):{Balance:new(big.Int).SetStrings(`14423000000000000000`,10)},
			common.HexToAddress(`fxbbf427b0fbe1f06317fe5bbc3b56204f3409d282`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fxbcc2db23e39974d55937365d849fa69a6e1abb02`):{Balance:new(big.Int).SetStrings(`632401236999999979520`,10)},
			common.HexToAddress(`fxbd0de08e856b9aae61cb87d290bf65fcc1c05cfa`):{Balance:new(big.Int).SetStrings(`300000000000000`,10)},
			common.HexToAddress(`fxbd34e7bc8e775bac052a14889ffa13917c0adae4`):{Balance:new(big.Int).SetStrings(`999979000000000000`,10)},
			common.HexToAddress(`fxbd37a64994945701ea8bfb215412f0d32fd2e0f3`):{Balance:new(big.Int).SetStrings(`14947900000000000000`,10)},
			common.HexToAddress(`fxbe5eb977b4d819221c092a0d537c564ef576661e`):{Balance:new(big.Int).SetStrings(`286279000000000000`,10)},
			common.HexToAddress(`fxbe7dc6a609969caab2a0e06b64431633524d2549`):{Balance:new(big.Int).SetStrings(`24601869000000000000`,10)},
			common.HexToAddress(`fxbf0fac531b2177475b2861010cfbbde13a25e989`):{Balance:new(big.Int).SetStrings(`49896558000000000000`,10)},
			common.HexToAddress(`fxbf39b7726f97a45e1eb29a0f4ab9f07cb6d76d8b`):{Balance:new(big.Int).SetStrings(`1222725353000000028672`,10)},
			common.HexToAddress(`fxbfe1af05ddbded1a8ed816133f1ca2d6ab5aba1e`):{Balance:new(big.Int).SetStrings(`131615540000000000000`,10)},
			common.HexToAddress(`fxc00781c7c6456c7aa1a41baa5c6a7a447ced16bc`):{Balance:new(big.Int).SetStrings(`10870600000000000000`,10)},
			common.HexToAddress(`fxc070abeb5a19089542fb07962eb1d259ed246cb0`):{Balance:new(big.Int).SetStrings(`203108857999999991808`,10)},
			common.HexToAddress(`fxc099d74b7a4e849390e1e294850fec7848532d2e`):{Balance:new(big.Int).SetStrings(`74212579000000004096`,10)},
			common.HexToAddress(`fxc147ac0eed8f8f938e72263500aae9fd6ee0ef01`):{Balance:new(big.Int).SetStrings(`241800000000000000`,10)},
			common.HexToAddress(`fxc16707281c038103962deb39fa7179148c29b1cb`):{Balance:new(big.Int).SetStrings(`221115400000000000000`,10)},
			common.HexToAddress(`fxc26622173837efa31bccf1aa37be139b87da99f1`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fxc2ab9931359af9fc416f63691af48d446d53ecd2`):{Balance:new(big.Int).SetStrings(`10870600000000000000`,10)},
			common.HexToAddress(`fxc3a576bbc7a323eed7673c435e8af50be955ab1c`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fxc3aa6516705fe881ff30d9830f031095cac09bbb`):{Balance:new(big.Int).SetStrings(`523929374000000008192`,10)},
			common.HexToAddress(`fxc3fc69bc747cf8fe1e80b80d73175dfddfff04f9`):{Balance:new(big.Int).SetStrings(`621311000000000000`,10)},
			common.HexToAddress(`fxc45bff32a1b6d22e3ff9e994030f8f7011260804`):{Balance:new(big.Int).SetStrings(`40706800000000000000`,10)},
			common.HexToAddress(`fxc48250cebf7f9f230115e5ab8e3b573af58c6ddf`):{Balance:new(big.Int).SetStrings(`48472300000000000000`,10)},
			common.HexToAddress(`fxc4de09bfa2ac3a181fc06fc603bfad6d453f5d64`):{Balance:new(big.Int).SetStrings(`3261990000000000000`,10)},
			common.HexToAddress(`fxc50ffc83af6d24f90930d61eb74bb0542b25712f`):{Balance:new(big.Int).SetStrings(`3235779000000000000`,10)},
			common.HexToAddress(`fxc537f253f5e3d0ed7eda2da9611dfdad6fea401f`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fxc63fe507f825018395956abbdd4f07b08c2a0fd1`):{Balance:new(big.Int).SetStrings(`113526260000000000000`,10)},
			common.HexToAddress(`fxc6843131dbbc35c4939b9789e6d3731fef57d7c1`):{Balance:new(big.Int).SetStrings(`33325200000000000000`,10)},
			common.HexToAddress(`fxc6b3cf821ee68a0b692781646f6e65c8255b077c`):{Balance:new(big.Int).SetStrings(`502442060000000016384`,10)},
			common.HexToAddress(`fxc826ae426840d2edcf588609081ade31beb6415f`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fxc879839d0fa453bb004ca51c55f96d201e2a2f14`):{Balance:new(big.Int).SetStrings(`999900000`,10)},
			common.HexToAddress(`fxc91c9a71cde152a68f383ac3144c51e6f6d0c49a`):{Balance:new(big.Int).SetStrings(`75027200000000000000`,10)},
			common.HexToAddress(`fxc945d7ceb98dc2aa2919ed9b19c2433cf952c166`):{Balance:new(big.Int).SetStrings(`43905800000000000000`,10)},
			common.HexToAddress(`fxc96e99a7dfc607d8a77ef1f310489d030dea9fe5`):{Balance:new(big.Int).SetStrings(`8214674000000000000`,10)},
			common.HexToAddress(`fxca07a2ee2728aef6543fc5d1fa02cdc19c44069e`):{Balance:new(big.Int).SetStrings(`2490279000000000000`,10)},
			common.HexToAddress(`fxcb062afad9a1c566556e67a68660e28b0dd7520c`):{Balance:new(big.Int).SetStrings(`26690000000000000`,10)},
			common.HexToAddress(`fxcb163d71f07783a8a8601b6fb4a4d33545c04364`):{Balance:new(big.Int).SetStrings(`1148479000000000000`,10)},
			common.HexToAddress(`fxcbb4769cea6411d08ed39b7de4d897fccca2bc8f`):{Balance:new(big.Int).SetStrings(`122417900000000000000`,10)},
			common.HexToAddress(`fxcbe9f50aa44f0fb60b3157658025da2334fecb18`):{Balance:new(big.Int).SetStrings(`1905554158000000204800`,10)},
			common.HexToAddress(`fxcc420b06ee86ccdde12a3e298c7c61b434f5a596`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fxcc45f128f4b7f5988ccb27cf9b746f4a05817965`):{Balance:new(big.Int).SetStrings(`9179000000000000`,10)},
			common.HexToAddress(`fxcc4c597b3fada8ffb6a51f66b17055cb4a841d06`):{Balance:new(big.Int).SetStrings(`10171376000000000000`,10)},
			common.HexToAddress(`fxcc55eb101fe659101c8f67957a06c7add7bc92cb`):{Balance:new(big.Int).SetStrings(`13338600000000000000`,10)},
			common.HexToAddress(`fxccbc06c725c1227973ebacb9a63ece4960b9be9b`):{Balance:new(big.Int).SetStrings(`9505000000000000000`,10)},
			common.HexToAddress(`fxcdea779c46cbf1a4c34cca504dd349b701a45cd7`):{Balance:new(big.Int).SetStrings(`20534800000000000000`,10)},
			common.HexToAddress(`fxce9c0af7f0bc0a59f5ba9ead696472b3ab8e2648`):{Balance:new(big.Int).SetStrings(`208585016000000000000`,10)},
			common.HexToAddress(`fxcede2c1ca6af052a48dcacadef59c50210e2f5a1`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fxd15aab7ccf8475f726509e6d383411904d49ff2b`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fxd180c67eaec708d5f72101d0eb24af55acde2a49`):{Balance:new(big.Int).SetStrings(`62760400000000000000`,10)},
			common.HexToAddress(`fxd247c1330653f115dc696da72433e7dfb7cc8c70`):{Balance:new(big.Int).SetStrings(`9790000000000000`,10)},
			common.HexToAddress(`fxd2a6ce6565891cf68d8279cb2617395b2f59b4a8`):{Balance:new(big.Int).SetStrings(`3365695000000000000`,10)},
			common.HexToAddress(`fxd2e4454dcdb871ff20f61bcd962f0fd82c4aa3c6`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fxd374ef25060ac1c941540f088f34244f0b41f7d9`):{Balance:new(big.Int).SetStrings(`173050657999999991808`,10)},
			common.HexToAddress(`fxd37d48a41e64b1b48c170df4284e935c6ab38ad9`):{Balance:new(big.Int).SetStrings(`204245178999999987712`,10)},
			common.HexToAddress(`fxd436131d5acea7370c183b430186dc9ea160b3c9`):{Balance:new(big.Int).SetStrings(`525358594999999987712`,10)},
			common.HexToAddress(`fxd4cfeee02ae7eb2ff044fefde3fabcdfe604c8fc`):{Balance:new(big.Int).SetStrings(`2893000000000000000`,10)},
			common.HexToAddress(`fxd5be73a063eaa339965786f215e558aef63c2eb7`):{Balance:new(big.Int).SetStrings(`246389800000000032768`,10)},
			common.HexToAddress(`fxd5f95ded560328c059743c512b844af0135ecdbc`):{Balance:new(big.Int).SetStrings(`1002724000000000000`,10)},
			common.HexToAddress(`fxd61f81cdd9d24bd46ef18b4a2f02982a58b434af`):{Balance:new(big.Int).SetStrings(`1790779000000000000`,10)},
			common.HexToAddress(`fxd62a3ad1ae5138a276ca303caa9489bbe04c8906`):{Balance:new(big.Int).SetStrings(`5782500000000000000`,10)},
			common.HexToAddress(`fxd635e1a504e4e6c5bf8dd313f901c7c6f90963f1`):{Balance:new(big.Int).SetStrings(`10765779000000000000`,10)},
			common.HexToAddress(`fxd6b8319637c74f7d9d2df0500be4b732525f244f`):{Balance:new(big.Int).SetStrings(`24265800000000000000`,10)},
			common.HexToAddress(`fxd7147415a846d9cdc0a89b4543cef5becfba62fe`):{Balance:new(big.Int).SetStrings(`49472300000000000000`,10)},
			common.HexToAddress(`fxd7e5f0c7c3e9df411d42053e460093cd9e4f4ef3`):{Balance:new(big.Int).SetStrings(`1748800000000000000`,10)},
			common.HexToAddress(`fxd7fca27b8ce786e6e3204c395cb6d70110c024bf`):{Balance:new(big.Int).SetStrings(`194839000000000000`,10)},
			common.HexToAddress(`fxd8264916dece51a4202afb2b5243d00412bdb87c`):{Balance:new(big.Int).SetStrings(`1243953686000000040960`,10)},
			common.HexToAddress(`fxd89bd68db0528a04953bc4ec098126328eecc7dd`):{Balance:new(big.Int).SetStrings(`108902660000000000000`,10)},
			common.HexToAddress(`fxd8b69246bb804e94964a8297249b0d732d7c610c`):{Balance:new(big.Int).SetStrings(`6669300000000000000`,10)},
			common.HexToAddress(`fxda04c4ec04d79dc99f3e18367fd762d1484ec552`):{Balance:new(big.Int).SetStrings(`885758000000000000`,10)},
			common.HexToAddress(`fxda0641c8f83c547155f211b61535a80c0be5734d`):{Balance:new(big.Int).SetStrings(`1132583400000000098304`,10)},
			common.HexToAddress(`fxdaf5b61245c3b78dcac4efbe75ec568070b7a590`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fxdb2751912681050c196ea905374f8847f5290a59`):{Balance:new(big.Int).SetStrings(`1345500000000000000`,10)},
			common.HexToAddress(`fxdb2e6a874ed68d1dc0210ffe35e260b66987123d`):{Balance:new(big.Int).SetStrings(`1748800000000000000`,10)},
			common.HexToAddress(`fxdb6d726ce3c99283b81eaacc9bfae4c1fceddb21`):{Balance:new(big.Int).SetStrings(`187208759999999967232`,10)},
			common.HexToAddress(`fxdb7bf96cc125644e57f49894fba044c38b49122e`):{Balance:new(big.Int).SetStrings(`100274436999999995904`,10)},
			common.HexToAddress(`fxdc18b750d64ef732b591acb86b50828b32a938cf`):{Balance:new(big.Int).SetStrings(`350109616000000000000`,10)},
			common.HexToAddress(`fxdc64f1bfd96fb8484055dc968df142d2ab4343de`):{Balance:new(big.Int).SetStrings(`41235800000000000000`,10)},
			common.HexToAddress(`fxdc956a8e6e356b2cdeb1ea1bc2d500ec5c6716a3`):{Balance:new(big.Int).SetStrings(`6482658000000000000`,10)},
			common.HexToAddress(`fxde7004a8bbaa8a559849ffb80f7664a30e2e73c8`):{Balance:new(big.Int).SetStrings(`51374479000000004096`,10)},
			common.HexToAddress(`fxdea7ae17083c7f189c4fb6369169c28b0d6a76e1`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fxdf00f7fc7b6767d874b75bc3844ea0787d737f71`):{Balance:new(big.Int).SetStrings(`16309300000000000000`,10)},
			common.HexToAddress(`fxdf471a5a717bd945fab93d5fc80dc5dc24e52006`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fxe12f777ec7212541d7ebd677866f4770e7785258`):{Balance:new(big.Int).SetStrings(`1544558000000000000`,10)},
			common.HexToAddress(`fxe202ffcd7455f696c15fe0bec1178ed18453d8e0`):{Balance:new(big.Int).SetStrings(`8148500000000000000`,10)},
			common.HexToAddress(`fxe208cea127c97446e1ba15f0a55f4e0084e0e9c9`):{Balance:new(big.Int).SetStrings(`623100395999999950848`,10)},
			common.HexToAddress(`fxe20ced2ecca4a050d4bae73adc8eca3d67eddbb7`):{Balance:new(big.Int).SetStrings(`64576574000000000000`,10)},
			common.HexToAddress(`fxe2efad4916e57274a5ef9c00a82d722ea0c6ae6e`):{Balance:new(big.Int).SetStrings(`25330600000000000000`,10)},
			common.HexToAddress(`fxe5c250a958f8152ea81747cc193b9863c98b02dc`):{Balance:new(big.Int).SetStrings(`23366416000000000000`,10)},
			common.HexToAddress(`fxe6abebf2969186d9edf4134a9a223f050e6a9a16`):{Balance:new(big.Int).SetStrings(`35998000000000000000`,10)},
			common.HexToAddress(`fxe6dcd6ad8b9c86e63576f6542972a3e95695dc74`):{Balance:new(big.Int).SetStrings(`504869547999999950848`,10)},
			common.HexToAddress(`fxe7034abaefdc31733b0a12ee0c0cf4b32ddc5e82`):{Balance:new(big.Int).SetStrings(`31044000000000000000`,10)},
			common.HexToAddress(`fxe7c220ea3c3c73b373f3e8f8e68689c5545cda65`):{Balance:new(big.Int).SetStrings(`28011000000000000`,10)},
			common.HexToAddress(`fxe7fbd89ed3683bb9ffc0be2fb1e71522764fc4b6`):{Balance:new(big.Int).SetStrings(`100209327000000004096`,10)},
			common.HexToAddress(`fxe81126f71fdc034a8bf29918908d803a308e7373`):{Balance:new(big.Int).SetStrings(`717560000000000000`,10)},
			common.HexToAddress(`fxe8a1cb2c8c6d1f2520305ce2de59dd836b58e943`):{Balance:new(big.Int).SetStrings(`10265779000000000000`,10)},
			common.HexToAddress(`fxe8ee34046e11d732a80f8bd5a66b1ed9d50b32db`):{Balance:new(big.Int).SetStrings(`525097315999999983616`,10)},
			common.HexToAddress(`fxe9172fbb30a3cfe4fbd41956d54570114b48b373`):{Balance:new(big.Int).SetStrings(`3239700000000000000`,10)},
			common.HexToAddress(`fxea4eef7d39c5abd178e9069f82982bfa4cbf7a7a`):{Balance:new(big.Int).SetStrings(`3148479000000000000`,10)},
			common.HexToAddress(`fxebaff6e5a4b3eeaea07fb0b530d1b23f357fe34d`):{Balance:new(big.Int).SetStrings(`430435000000000032768`,10)},
			common.HexToAddress(`fxebf49fffc55c7e239866bb0dd6391d3249e34702`):{Balance:new(big.Int).SetStrings(`204814279000000004096`,10)},
			common.HexToAddress(`fxec49e89c6d125e9b6411f3e09609709ff9396d69`):{Balance:new(big.Int).SetStrings(`416302736999999995904`,10)},
			common.HexToAddress(`fxeca7f2595412b59fa4964bdc77358cb3c5dff2bb`):{Balance:new(big.Int).SetStrings(`105738300000000000000`,10)},
			common.HexToAddress(`fxecc892d30f96844574ce18c5a4361f0fc487cb14`):{Balance:new(big.Int).SetStrings(`1000000000000000000`,10)},
			common.HexToAddress(`fxefcc40417c590351169bd26d70090be6d7812640`):{Balance:new(big.Int).SetStrings(`106243000000000000`,10)},
			common.HexToAddress(`fxf0f50033ab3dd060b13b5ea7743b66b5e06e4786`):{Balance:new(big.Int).SetStrings(`6045600000000000000`,10)},
			common.HexToAddress(`fxf199ea09f1bfffa655f9d388cd7d23857c223a0c`):{Balance:new(big.Int).SetStrings(`5092279000000000000`,10)},
			common.HexToAddress(`fxf1fb6ebeb9f4afe8fa2fdf90b6062b3b9941942f`):{Balance:new(big.Int).SetStrings(`104490958000000008192`,10)},
			common.HexToAddress(`fxf21605b8834720b40d65096e871ddb1924761a33`):{Balance:new(big.Int).SetStrings(`37544600000000000000`,10)},
			common.HexToAddress(`fxf319db3a61ca1111c06b65b9d4538e13ad7cfa71`):{Balance:new(big.Int).SetStrings(`41235800000000000000`,10)},
			common.HexToAddress(`fxf45357d92da79bf51b1b1584dbe15d8782a56eba`):{Balance:new(big.Int).SetStrings(`180765100000000016384`,10)},
			common.HexToAddress(`fxf510fd7bfcf2bacda40349ace59d8e8e7b06fd1e`):{Balance:new(big.Int).SetStrings(`120900000000000000`,10)},
			common.HexToAddress(`fxf598f754cbdc3b27757951437ea38656c2c244a6`):{Balance:new(big.Int).SetStrings(`172123500000000016384`,10)},
			common.HexToAddress(`fxf5ba4e74f33465b5610bd1100b8f4e0f4e115371`):{Balance:new(big.Int).SetStrings(`1570158000000000000`,10)},
			common.HexToAddress(`fxf744afb707a3693922c111767c543ba172ed5a4e`):{Balance:new(big.Int).SetStrings(`3239700000000000000`,10)},
			common.HexToAddress(`fxf7527912c359887b86167c2bdb79a56d12d8ec43`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fxf857faec39234bc1aa0648bf46a511754fc03e8f`):{Balance:new(big.Int).SetStrings(`41235800000000000000`,10)},
			common.HexToAddress(`fxf88db5fe7075a7bd7ad75487a1f9cd3315aa3833`):{Balance:new(big.Int).SetStrings(`5480400000000000000`,10)},
			common.HexToAddress(`fxf8b028532b6c378c8453201a694a4bc4b1846fe4`):{Balance:new(big.Int).SetStrings(`56705900000000000000`,10)},
			common.HexToAddress(`fxf8fa646f332149dd8edc639b8affc723889053bd`):{Balance:new(big.Int).SetStrings(`418486495000000004096`,10)},
			common.HexToAddress(`fxf95f5b2a79daf1efdd5e62f4b20f8859368f6932`):{Balance:new(big.Int).SetStrings(`2000000000000000000`,10)},
			common.HexToAddress(`fxf981b18b713b7224a6754a1aab6ea6104a626d7e`):{Balance:new(big.Int).SetStrings(`14423000000000000000`,10)},
			common.HexToAddress(`fxfa95938fcaa8c2eb55048ff8beca2d9de29413aa`):{Balance:new(big.Int).SetStrings(`115757900000000000000`,10)},
			common.HexToAddress(`fxfb055fa443da0bc23e02e429670071e73838a6b6`):{Balance:new(big.Int).SetStrings(`86354610999999987712`,10)},
			common.HexToAddress(`fxfb87541e5ea2b71ee2ccc6f6de241dc2efc40c49`):{Balance:new(big.Int).SetStrings(`2734600000000000000`,10)},
			common.HexToAddress(`fxfb97b4a97f9b7654c96bedbedcea373878360603`):{Balance:new(big.Int).SetStrings(`16662600000000000000`,10)},
			common.HexToAddress(`fxfbbc64737e1595ee9030fbd1d83faee8c068e968`):{Balance:new(big.Int).SetStrings(`32414300000000000000`,10)},
			common.HexToAddress(`fxfc3e853d08918cad049e42eb6dc78e8abbc170e8`):{Balance:new(big.Int).SetStrings(`313156637000000012288`,10)},
			common.HexToAddress(`fxfc559f8af4f279e0ffb07d8706203ed65bb52708`):{Balance:new(big.Int).SetStrings(`4822597685000000241664`,10)},
			common.HexToAddress(`fxfd298705abca4df4d6720d7e3bf60425ee7e8e23`):{Balance:new(big.Int).SetStrings(`1362400000000000000`,10)},
			common.HexToAddress(`fxfd81eef7906c9d014f07e1b04ffdf6492a1f19da`):{Balance:new(big.Int).SetStrings(`4096500000000000000`,10)},
			common.HexToAddress(`fxff72633a10606c710d11711098c9f5005c436172`):{Balance:new(big.Int).SetStrings(`9505000000000000000`,10)},
			common.HexToAddress(`fxffe81f86a0f44c18046d6dff2399fd02f0a20af9`):{Balance:new(big.Int).SetStrings(`2268058921999999959040`,10)},

		},
	}
}

// DefaultRopstenGenesisBlock returns the Ropsten network genesis block.
func DefaultRopstenGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.RopstenChainConfig,
		Nonce:      66,
		ExtraData:  hexutil.MustDecode("0x3535353535353535353535353535353535353535353535353535353535353535"),
		GasLimit:   16777216,
		Difficulty: big.NewInt(1),
		Alloc:      decodePrealloc(ropstenAllocData),
	}
}

// DefaultRinkebyGenesisBlock returns the Rinkeby network genesis block.
func DefaultRinkebyGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.RinkebyChainConfig,
		Timestamp:  1492009146,
		ExtraData:  hexutil.MustDecode("0x52657370656374206d7920617574686f7269746168207e452e436172746d616e42eb768f2244c8811c63729a21a3569731535f067ffc57839b00206d1ad20c69a1981b489f772031b279182d99e65703f0076e4812653aab85fca0f00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   4700000,
		Difficulty: big.NewInt(1),
		Alloc:      decodePrealloc(rinkebyAllocData),
	}
}

// DefaultGoerliGenesisBlock returns the Görli network genesis block.
func DefaultGoerliGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.GoerliChainConfig,
		Timestamp:  1548854791,
		ExtraData:  hexutil.MustDecode("0x22466c6578692069732061207468696e6722202d204166726900000000000000e0a2bd4258d2768837baa26a28fe71dc079f84c70000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   10485760,
		Difficulty: big.NewInt(1),
		Alloc:      decodePrealloc(goerliAllocData),
	}
}

func DefaultYoloV2GenesisBlock() *Genesis {
	// TODO: Update with yolov2 values + regenerate alloc data
	return &Genesis{
		Config:     params.YoloV2ChainConfig,
		Timestamp:  0x5f91b932,
		ExtraData:  hexutil.MustDecode("0x00000000000000000000000000000000000000000000000000000000000000008a37866fd3627c9205a37c8685666f32ec07bb1b0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   0x47b760,
		Difficulty: big.NewInt(1),
		Alloc:      decodePrealloc(yoloV1AllocData),
	}
}

// DeveloperGenesisBlock returns the 'geth --dev' genesis block.
func DeveloperGenesisBlock(period uint64, faucet common.Address) *Genesis {
	// Override the default period to the user requested one
	config := *params.AllCliqueProtocolChanges
	config.Clique.Period = period

	// Assemble and return the genesis with the precompiles and faucet pre-funded
	return &Genesis{
		Config:     &config,
		ExtraData:  append(append(make([]byte, 32), faucet[:]...), make([]byte, crypto.SignatureLength)...),
		GasLimit:   11500000,
		Difficulty: big.NewInt(1),
		Alloc: map[common.Address]GenesisAccount{
			common.BytesToAddress([]byte{1}): {Balance: big.NewInt(1)}, // ECRecover
			common.BytesToAddress([]byte{2}): {Balance: big.NewInt(1)}, // SHA256
			common.BytesToAddress([]byte{3}): {Balance: big.NewInt(1)}, // RIPEMD
			common.BytesToAddress([]byte{4}): {Balance: big.NewInt(1)}, // Identity
			common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
			common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
			common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
			common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
			common.BytesToAddress([]byte{9}): {Balance: big.NewInt(1)}, // BLAKE2b
			faucet:                           {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
		},
	}
}

func decodePrealloc(data string) GenesisAlloc {
	var p []struct{ Addr, Balance *big.Int }
	if err := rlp.NewStream(strings.NewReader(data), 0).Decode(&p); err != nil {
		panic(err)
	}
	ga := make(GenesisAlloc, len(p))
	for _, account := range p {
		ga[common.BigToAddress(account.Addr)] = GenesisAccount{Balance: account.Balance}
	}
	return ga
}
