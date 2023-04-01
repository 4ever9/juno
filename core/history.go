package core

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/juno/db"
)

type History struct {
	txn db.Transaction
}

func NewHistory(txn db.Transaction) *History {
	return &History{txn: txn}
}

func logDBKey(key []byte, height uint64) []byte {
	return binary.BigEndian.AppendUint64(key, height)
}

func (h *History) logOldValue(key, value []byte, height uint64) error {
	return h.txn.Set(logDBKey(key, height), value)
}

func (h *History) deleteLog(key []byte, height uint64) error {
	return h.txn.Delete(logDBKey(key, height))
}

func (h *History) valueAt(key []byte, height uint64) ([]byte, error) {
	it, err := h.txn.NewIterator()
	if err != nil {
		return nil, err
	}

	for it.Seek(logDBKey(key, height)); it.Valid(); it.Next() {
		seekedKey := it.Key()
		// seekedKey size should be `len(key) + sizeof(uint64)` and seekedKey should match key prefix
		if len(seekedKey) != len(key)+8 || !bytes.HasPrefix(seekedKey, key) {
			break
		}

		seekedHeight := binary.BigEndian.Uint64(seekedKey[len(key):])
		if seekedHeight < height {
			// last change happened before the height we are looking for
			// check head state
			break
		} else if seekedHeight == height {
			// a log exists for the height we are looking for, so the old value in this log entry is not useful.
			// advance the iterator and see we can use the next entry. If not, ErrCheckHeadState will be returned
			continue
		}

		val, itErr := it.Value()
		if err = db.CloseAndWrapOnError(it.Close, itErr); err != nil {
			return nil, err
		}
		// seekedHeight > height
		return val, nil
	}

	return nil, db.CloseAndWrapOnError(it.Close, errors.New("check head state"))
}

func storageLogKey(contractAddress, storageLocation *felt.Felt) []byte {
	return db.ContractStorageHistory.Key(contractAddress.Marshal(), storageLocation.Marshal())
}

// LogContractStorage logs the old value of a storage location for the given contract which changed on height `height`
func (h *History) LogContractStorage(contractAddress, storageLocation, oldValue *felt.Felt, height uint64) error {
	key := storageLogKey(contractAddress, storageLocation)
	return h.logOldValue(key, oldValue.Marshal(), height)
}

// DeleteContractStorageLog deletes the log at the given height
func (h *History) DeleteContractStorageLog(contractAddress, storageLocation *felt.Felt, height uint64) error {
	return h.deleteLog(storageLogKey(contractAddress, storageLocation), height)
}

// ContractStorageAt returns the value of a storage location of the given contract at the height `height`
func (h *History) ContractStorageAt(contractAddress, storageLocation *felt.Felt, height uint64) (*felt.Felt, error) {
	key := storageLogKey(contractAddress, storageLocation)
	value, err := h.valueAt(key, height)
	if err != nil {
		return nil, err
	}

	return new(felt.Felt).SetBytes(value), nil
}

func nonceLogKey(contractAddress *felt.Felt) []byte {
	return db.ContractNonceHistory.Key(contractAddress.Marshal())
}

func (h *History) LogContractNonce(contractAddress, oldValue *felt.Felt, height uint64) error {
	return h.logOldValue(nonceLogKey(contractAddress), oldValue.Marshal(), height)
}

func (h *History) DeleteContractNonceLog(contractAddress *felt.Felt, height uint64) error {
	return h.deleteLog(nonceLogKey(contractAddress), height)
}

func (h *History) ContractNonceAt(contractAddress *felt.Felt, height uint64) (*felt.Felt, error) {
	key := nonceLogKey(contractAddress)
	value, err := h.valueAt(key, height)
	if err != nil {
		return nil, err
	}

	return new(felt.Felt).SetBytes(value), nil
}

func classHashLogKey(contractAddress *felt.Felt) []byte {
	return db.ContractClassHashHistory.Key(contractAddress.Marshal())
}

func (h *History) LogContractClassHash(contractAddress, oldValue *felt.Felt, height uint64) error {
	return h.logOldValue(classHashLogKey(contractAddress), oldValue.Marshal(), height)
}

func (h *History) DeleteContractClassHashLog(contractAddress *felt.Felt, height uint64) error {
	return h.deleteLog(classHashLogKey(contractAddress), height)
}

func (h *History) ContractClassHashAt(contractAddress *felt.Felt, height uint64) (*felt.Felt, error) {
	key := classHashLogKey(contractAddress)
	value, err := h.valueAt(key, height)
	if err != nil {
		return nil, err
	}

	return new(felt.Felt).SetBytes(value), nil
}
