package marker

import (
	"sort"
	"strconv"

	"github.com/iotaledger/hive.go/byteutils"
	"github.com/iotaledger/hive.go/cerrors"
	"github.com/iotaledger/hive.go/marshalutil"
	"github.com/iotaledger/hive.go/objectstorage"
	"github.com/mr-tron/base58"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/xerrors"
)

// region Sequence /////////////////////////////////////////////////////////////////////////////////////////////////////

// Sequence represents a marker sequence.
type Sequence struct {
	id               SequenceID
	parentReferences ParentReferences
	rank             uint64
	highestIndex     Index

	objectstorage.StorableObjectFlags
}

// New creates a new Sequence.
func NewSequence(id SequenceID, referencedMarkers Markers, rank uint64) *Sequence {
	return &Sequence{
		id:               id,
		parentReferences: NewParentReferences(referencedMarkers),
		rank:             rank,
		highestIndex:     referencedMarkers.HighestIndex() + 1,
	}
}

// SequenceFromBytes unmarshals a Sequence from a sequence of bytes.
func SequenceFromBytes(sequenceBytes []byte) (sequence *Sequence, consumedBytes int, err error) {
	marshalUtil := marshalutil.New(sequenceBytes)
	if sequence, err = SequenceFromMarshalUtil(marshalUtil); err != nil {
		err = xerrors.Errorf("failed to parse Sequence from MarshalUtil: %w", err)
		return
	}
	consumedBytes = marshalUtil.ReadOffset()

	return
}

// SequenceFromMarshalUtil is a wrapper for simplified unmarshaling in a byte stream using the marshalUtil package.
func SequenceFromMarshalUtil(marshalUtil *marshalutil.MarshalUtil) (sequence *Sequence, err error) {
	sequence = &Sequence{}
	if sequence.id, err = SequenceIDFromMarshalUtil(marshalUtil); err != nil {
		err = xerrors.Errorf("failed to parse SequenceID from MarshalUtil: %w", err)
		return
	}
	if sequence.parentReferences, err = ParentReferencesFromMarshalUtil(marshalUtil); err != nil {
		err = xerrors.Errorf("failed to parse ParentReferences from MarshalUtil: %w", err)
		return
	}
	if sequence.rank, err = marshalUtil.ReadUint64(); err != nil {
		err = xerrors.Errorf("failed to parse rank (%v): %w", err, cerrors.ErrParseBytesFailed)
		return
	}
	if sequence.highestIndex, err = IndexFromMarshalUtil(marshalUtil); err != nil {
		err = xerrors.Errorf("failed to parse highest Index from MarshalUtil: %w", err)
		return
	}

	return
}

// SequenceFromObjectStorage restores an Sequence that was stored in the ObjectStorage.
func SequenceFromObjectStorage(key []byte, data []byte) (sequence objectstorage.StorableObject, err error) {
	if sequence, _, err = SequenceFromBytes(byteutils.ConcatBytes(key, data)); err != nil {
		err = xerrors.Errorf("failed to parse Sequence from bytes: %w", err)
		return
	}

	return
}

// ID returns the id of the marker sequence.
func (s *Sequence) ID() SequenceID {
	return s.id
}

// ParentReferences returns the sequence ids of the parent sequences.
func (s *Sequence) ParentSequences() SequenceIDs {
	return s.parentReferences.SequenceIDs()
}

// HighestReferencedParentMarkers returns a list of highest index markers in different marker sequence of parent sequences.
func (s *Sequence) HighestReferencedParentMarkers(index Index) UniqueMarkers {
	return s.parentReferences.HighestReferencedMarkers(index)
}

// Rank returns the rank of the sequence.
func (s *Sequence) Rank() uint64 {
	return s.rank
}

// HighestIndex returns the highest index of the sequence.
func (s *Sequence) HighestIndex() Index {
	return s.highestIndex
}

// Bytes returns the Sequence in serialized byte form.
func (s *Sequence) Bytes() []byte {
	return byteutils.ConcatBytes(s.ObjectStorageKey(), s.ObjectStorageValue())
}

// Update updates the sequence to object storage.
func (s *Sequence) Update(other objectstorage.StorableObject) {
	panic("updates disabled")
}

// ObjectStorageKey returns the key that is used to store the object in the database. It is required to match the
// StorableObject interface.
func (s *Sequence) ObjectStorageKey() []byte {
	return s.id.Bytes()
}

// ObjectStorageValue marshals the Sequence into a sequence of bytes. The ID is not serialized here as it is only used as
// a key in the ObjectStorage.
func (s *Sequence) ObjectStorageValue() []byte {
	return marshalutil.New().
		Write(s.parentReferences).
		WriteUint64(s.rank).
		Write(s.HighestIndex()).
		Bytes()
}

var _ objectstorage.StorableObject = &Sequence{}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region CachedSequence ///////////////////////////////////////////////////////////////////////////////////////////////

// CachedSequence is a wrapper for the generic CachedObject returned by the objectstorage that
// overrides the accessor methods with a type-casted one.
type CachedSequence struct {
	objectstorage.CachedObject
}

// Retain marks this CachedObject to still be in use by the program.
func (c *CachedSequence) Retain() *CachedSequence {
	return &CachedSequence{c.CachedObject.Retain()}
}

// Unwrap is the type-casted equivalent of Get. It returns nil if the object does not exist.
func (c *CachedSequence) Unwrap() *Sequence {
	untypedObject := c.Get()
	if untypedObject == nil {
		return nil
	}

	typedObject := untypedObject.(*Sequence)
	if typedObject == nil || typedObject.IsDeleted() {
		return nil
	}

	return typedObject
}

// Consume unwraps the CachedObject and passes a type-casted version to the consumer. It automatically releases the
// object when the consumer finishes and returns true of there was at least one object that was consumed.
func (c *CachedSequence) Consume(consumer func(sequence *Sequence), forceRelease ...bool) (consumed bool) {
	return c.CachedObject.Consume(func(object objectstorage.StorableObject) {
		consumer(object.(*Sequence))
	}, forceRelease...)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region SequenceID ///////////////////////////////////////////////////////////////////////////////////////////////////

// SequenceID identifies a marker sequence.
type SequenceID uint64

// SequenceIDFromBytes unmarshals a sequence ID from a sequence of bytes.
func SequenceIDFromBytes(sequenceIDBytes []byte) (sequenceID SequenceID, consumedBytes int, err error) {
	marshalUtil := marshalutil.New(sequenceIDBytes)
	if sequenceID, err = SequenceIDFromMarshalUtil(marshalUtil); err != nil {
		err = xerrors.Errorf("failed to parse SequenceID from MarshalUtil: %w", err)
		return
	}
	consumedBytes = marshalUtil.ReadOffset()

	return
}

// SequenceIDFromMarshalUtil is a wrapper for simplified unmarshaling in a byte stream using the marshalUtil package.
func SequenceIDFromMarshalUtil(marshalUtil *marshalutil.MarshalUtil) (sequenceID SequenceID, err error) {
	untypedSequenceID, err := marshalUtil.ReadUint64()
	if err != nil {
		err = xerrors.Errorf("failed to parse SequenceID (%v): %w", err, cerrors.ErrParseBytesFailed)
		return
	}
	sequenceID = SequenceID(untypedSequenceID)

	return
}

// Bytes returns the bytes of the sequence ID.
func (a SequenceID) Bytes() []byte {
	return marshalutil.New(marshalutil.Uint16Size).WriteUint64(uint64(a)).Bytes()
}

// String returns the base58 encode of the SequenceID.
func (a SequenceID) String() string {
	return "SequenceID(" + strconv.FormatUint(uint64(a), 10) + ")"
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region SequenceIDs //////////////////////////////////////////////////////////////////////////////////////////////////

// SequenceIDs represents a list of sequence IDs.
type SequenceIDs []SequenceID

// NewSequenceIDs create a new SequenceIDs.
func NewSequenceIDs(sequenceIDs ...SequenceID) (result SequenceIDs) {
	sort.Slice(sequenceIDs, func(i, j int) bool { return sequenceIDs[i] < sequenceIDs[j] })
	result = make(SequenceIDs, len(sequenceIDs))
	for i, sequenceID := range sequenceIDs {
		result[i] = sequenceID
	}

	return
}

// SequenceIDsFromBytes unmarshals a collection of sequence IDs from a sequence of bytes.
func SequenceIDsFromBytes(sequenceIDBytes []byte) (sequenceIDs SequenceIDs, consumedBytes int, err error) {
	marshalUtil := marshalutil.New(sequenceIDBytes)
	if sequenceIDs, err = SequenceIDsFromMarshalUtil(marshalUtil); err != nil {
		err = xerrors.Errorf("failed to parse SequenceIDs from MarshalUtil: %w", err)
		return
	}
	consumedBytes = marshalUtil.ReadOffset()

	return
}

// SequenceIDsFromMarshalUtil unmarshals a collection of Sequence IDs using a MarshalUtil (for easier unmarshaling).
func SequenceIDsFromMarshalUtil(marshalUtil *marshalutil.MarshalUtil) (sequenceIDs SequenceIDs, err error) {
	sequenceIDsCount, err := marshalUtil.ReadUint32()
	if err != nil {
		err = xerrors.Errorf("failed to parse SequenceIDs count (%v): %w", err, cerrors.ErrParseBytesFailed)
		return
	}
	sequenceIDs = make(SequenceIDs, sequenceIDsCount)
	for i := uint32(0); i < sequenceIDsCount; i++ {
		if sequenceIDs[i], err = SequenceIDFromMarshalUtil(marshalUtil); err != nil {
			err = xerrors.Errorf("failed to parse SequenceID from MarshalUtil: %w", err)
			return
		}
	}

	return
}

// SequenceAlias returns a SequenceAlias computed from SequenceIDs.
func (s SequenceIDs) SequenceAlias() (aggregatedSequencesID SequenceAlias) {
	marshalUtil := marshalutil.New(marshalutil.Uint64Size * len(s))
	for sequenceID := range s {
		marshalUtil.WriteUint64(uint64(sequenceID))
	}
	aggregatedSequencesID = blake2b.Sum256(marshalUtil.Bytes())

	return
}

// Bytes returns the SequenceIDs in serialized byte form.
func (s SequenceIDs) Bytes() []byte {
	marshalUtil := marshalutil.New()
	marshalUtil.WriteUint32(uint32(len(s)))
	for _, sequenceID := range s {
		marshalUtil.Write(sequenceID)
	}

	return marshalUtil.Bytes()
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region SequenceAlias ////////////////////////////////////////////////////////////////////////////////////////////////

// SequenceAliasLength defines length of an alias sequence ID.
const SequenceAliasLength = 32

// SequenceAlias identifies an alias sequence ID.
type SequenceAlias [SequenceAliasLength]byte

// SequenceAliasFromBytes unmarshals a sequence alias from a sequence of bytes.
func SequenceAliasFromBytes(aggregatedSequencesIDBytes []byte) (aggregatedSequencesID SequenceAlias, consumedBytes int, err error) {
	marshalUtil := marshalutil.New(aggregatedSequencesIDBytes)
	if aggregatedSequencesID, err = SequenceAliasFromMarshalUtil(marshalUtil); err != nil {
		err = xerrors.Errorf("failed to parse SequenceAlias from MarshalUtil: %w", err)
		return
	}
	consumedBytes = marshalUtil.ReadOffset()

	return
}

// SequenceAliasFromBase58 creates a SequenceAlias from a base58 encoded string.
func SequenceAliasFromBase58(base58String string) (aggregatedSequencesID SequenceAlias, err error) {
	bytes, err := base58.Decode(base58String)
	if err != nil {
		err = xerrors.Errorf("error while decoding base58 encoded SequenceAlias (%v): %w", err, cerrors.ErrBase58DecodeFailed)
		return
	}

	if aggregatedSequencesID, _, err = SequenceAliasFromBytes(bytes); err != nil {
		err = xerrors.Errorf("failed to parse SequenceAlias from bytes: %w", err)
		return
	}

	return
}

// SequenceAliasFromMarshalUtil unmarshals a SequenceAlias using a MarshalUtil (for easier unmarshaling).
func SequenceAliasFromMarshalUtil(marshalUtil *marshalutil.MarshalUtil) (aggregatedSequencesID SequenceAlias, err error) {
	aggregatedSequencesIDBytes, err := marshalUtil.ReadBytes(SequenceAliasLength)
	if err != nil {
		err = xerrors.Errorf("failed to parse SequenceAlias (%v): %w", err, cerrors.ErrParseBytesFailed)
		return
	}
	copy(aggregatedSequencesID[:], aggregatedSequencesIDBytes)

	return
}

// Bytes returns the bytes of the SequenceAlias.
func (a SequenceAlias) Bytes() []byte {
	return a[:]
}

// Base58 returns a base58 encoded version of the SequenceAlias.
func (a SequenceAlias) Base58() string {
	return base58.Encode(a.Bytes())
}

// String creates a human readable version of the SequenceAlias.
func (a SequenceAlias) String() string {
	return "SequenceAlias(" + a.Base58() + ")"
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region SequenceAliasMapping /////////////////////////////////////////////////////////////////////////////////////////

// SequenceAliasMapping represents a payload that executes a value transfer in the ledger state.
type SequenceAliasMapping struct {
	sequenceAlias SequenceAlias
	sequenceID    SequenceID

	objectstorage.StorableObjectFlags
}

// SequenceAliasMappingFromBytes unmarshals a SequenceAliasMapping from a sequence of bytes.
func SequenceAliasMappingFromBytes(mappingBytes []byte) (mapping *SequenceAliasMapping, consumedBytes int, err error) {
	marshalUtil := marshalutil.New(mappingBytes)
	if mapping, err = SequenceAliasMappingFromMarshalUtil(marshalUtil); err != nil {
		err = xerrors.Errorf("failed to parse SequenceAliasMapping from MarshalUtil: %w", err)
		return
	}
	consumedBytes = marshalUtil.ReadOffset()

	return
}

// SequenceAliasMappingFromMarshalUtil unmarshals a SequenceAliasMapping using a MarshalUtil (for easier unmarshaling).
func SequenceAliasMappingFromMarshalUtil(marshalUtil *marshalutil.MarshalUtil) (mapping *SequenceAliasMapping, err error) {
	mapping = &SequenceAliasMapping{}
	if mapping.sequenceAlias, err = SequenceAliasFromMarshalUtil(marshalUtil); err != nil {
		err = xerrors.Errorf("failed to parse SequenceAlias from MarshalUtil: %w", err)
		return
	}
	if mapping.sequenceID, err = SequenceIDFromMarshalUtil(marshalUtil); err != nil {
		err = xerrors.Errorf("failed to parse SequenceID from MarshalUtil: %w", err)
		return
	}

	return
}

// SequenceAliasMappingFromObjectStorage restores a SequenceAlias that was stored in the ObjectStorage.
func SequenceAliasMappingFromObjectStorage(key []byte, data []byte) (mapping objectstorage.StorableObject, err error) {
	if mapping, _, err = SequenceAliasMappingFromBytes(data); err != nil {
		err = xerrors.Errorf("failed to parse SequenceAliasMapping from bytes: %w", err)
		return
	}

	return
}

// SequenceAlias returns the SequenceAlias of SequenceAliasMapping.
func (a *SequenceAliasMapping) SequenceAlias() SequenceAlias {
	return a.sequenceAlias
}

// SequenceID returns the sequence ID of SequenceAliasMapping.
func (a *SequenceAliasMapping) SequenceID() SequenceID {
	return a.sequenceID
}

// Bytes returns a marshaled version of the SequenceAliasMapping.
func (a *SequenceAliasMapping) Bytes() []byte {
	return byteutils.ConcatBytes(a.ObjectStorageKey(), a.ObjectStorageValue())
}

// Update updates the SequenceAliasMapping to object storage.
func (a *SequenceAliasMapping) Update(other objectstorage.StorableObject) {
	panic("updates disabled")
}

// ObjectStorageKey returns the key that is used to store the object in the database. It is required to match the
// StorableObject interface.
func (a *SequenceAliasMapping) ObjectStorageKey() []byte {
	return a.sequenceAlias.Bytes()
}

// ObjectStorageValue marshals the Transaction into a sequence of bytes. The ID is not serialized here as it is only
// used as a key in the ObjectStorage.
func (a *SequenceAliasMapping) ObjectStorageValue() []byte {
	return a.sequenceID.Bytes()
}

var _ objectstorage.StorableObject = &SequenceAliasMapping{}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region CachedSequenceAliasMapping ///////////////////////////////////////////////////////////////////////////////////

// CachedSequenceAliasMapping is a wrapper for the generic CachedObject returned by the objectstorage that overrides the
// accessor methods with a type-casted one.
type CachedSequenceAliasMapping struct {
	objectstorage.CachedObject
}

// Retain marks this CachedObject to still be in use by the program.
func (c *CachedSequenceAliasMapping) Retain() *CachedSequenceAliasMapping {
	return &CachedSequenceAliasMapping{c.CachedObject.Retain()}
}

// Unwrap is the type-casted equivalent of Get. It returns nil if the object does not exist.
func (c *CachedSequenceAliasMapping) Unwrap() *SequenceAliasMapping {
	untypedObject := c.Get()
	if untypedObject == nil {
		return nil
	}

	typedObject := untypedObject.(*SequenceAliasMapping)
	if typedObject == nil || typedObject.IsDeleted() {
		return nil
	}

	return typedObject
}

// Consume unwraps the CachedObject and passes a type-casted version to the consumer. It automatically releases the
// object when the consumer finishes and returns true of there was at least one object that was consumed.
func (c *CachedSequenceAliasMapping) Consume(consumer func(aggregatedSequencesIDMapping *SequenceAliasMapping), forceRelease ...bool) (consumed bool) {
	return c.CachedObject.Consume(func(object objectstorage.StorableObject) {
		consumer(object.(*SequenceAliasMapping))
	}, forceRelease...)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
