package rtree

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
)

var magicNumber = [4]byte{'r', 't', 'r', 'f'} // https://gist.github.com/leommoore/f9e57ba2aa4bf197ebc5

const (
	SoftwareVersionMajor uint16 = 0
	SoftwareVersionMinor uint16 = 0
	SoftwareVersionPatch uint16 = 1
)

// Coordinate type IDs stored in header[10]
const (
	coordCustom  uint8 = 0
	coordInt32   uint8 = 1
	coordFloat64 uint8 = 2
	coordFloat32 uint8 = 3
	coordInt64   uint8 = 4
	coordUint32  uint8 = 5
	coordUint64  uint8 = 6
	coordInt     uint8 = 7
	coordUint    uint8 = 8
	coordInt16   uint8 = 9
	coordUint16  uint8 = 10
	coordInt8    uint8 = 11
	coordUint8   uint8 = 12
	coordUintptr uint8 = 13
)

const (
	// 1-byte types
	int8Size  int = 1
	uint8Size int = 1

	// 2-byte types
	int16Size  int = 2
	uint16Size int = 2

	// 4-byte types
	int32Size   int = 4
	uint32Size  int = 4
	float32Size int = 4

	// 8-byte fixed types
	int64Size   int = 8
	uint64Size  int = 8
	float64Size int = 8

	intSize     int = 8
	uintSize    int = 8
	uintptrSize int = 8
)

func getCoordEncoder[N numeric]() (int, uint8, func([]byte, N), error) {
	var zero N
	switch any(zero).(type) {
	case int32:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint32(b, uint32(v))
		}
		return int32Size, coordInt32, f, nil

	case float64:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint64(b, math.Float64bits(float64(v)))
		}
		return float64Size, coordFloat64, f, nil

	case float32:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint32(b, math.Float32bits(float32(v)))
		}
		return float32Size, coordFloat32, f, nil

	case int64:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint64(b, uint64(v))
		}
		return int64Size, coordInt64, f, nil
	case uint32:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint32(b, uint32(v))
		}
		return uint32Size, coordUint32, f, nil

	case uint64:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint64(b, uint64(v))
		}
		return uint64Size, coordUint64, f, nil

	case int:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint64(b, uint64(int64(v)))
		}
		return intSize, coordInt, f, nil

	case uint:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint64(b, uint64(v))
		}
		return uintSize, coordUint, f, nil
	case int16:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint16(b, uint16(v))
		}
		return int16Size, coordInt16, f, nil

	case uint16:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint16(b, uint16(v))
		}
		return uint16Size, coordUint16, f, nil

	case int8:
		f := func(b []byte, v N) {
			b[0] = byte(v)
		}
		return int8Size, coordInt8, f, nil

	case uint8:
		f := func(b []byte, v N) {
			b[0] = byte(v)
		}
		return uint8Size, coordUint8, f, nil

	case uintptr:
		f := func(b []byte, v N) {
			binary.LittleEndian.PutUint64(b, uint64(v))
		}
		return uintptrSize, coordUintptr, f, nil
	default:
		return 0, 0, nil, fmt.Errorf("unsupported coordinate numeric type: %T", zero)
	}
}

func getCoordDecoder[N numeric](fileTypeId uint8) (int, func([]byte) N, error) {
	_, typeId, _, err := getCoordEncoder[N]()
	if err != nil {
		return 0, nil, err
	}
	if fileTypeId != typeId {
		return 0, nil, fmt.Errorf("mismatched coordinate type ID: file has %d, tree expects %d", fileTypeId, typeId)
	}

	var zero N
	switch any(zero).(type) {
	case int32:
		f := func(b []byte) N {
			return N(int32(binary.LittleEndian.Uint32(b)))
		}
		return int32Size, f, nil

	case float64:
		f := func(b []byte) N {
			return N(math.Float64frombits(binary.LittleEndian.Uint64(b)))
		}
		return float64Size, f, nil

	case float32:
		f := func(b []byte) N {
			return N(math.Float32frombits(binary.LittleEndian.Uint32(b)))
		}
		return float32Size, f, nil

	case int64:
		f := func(b []byte) N {
			return N(int64(binary.LittleEndian.Uint64(b)))
		}
		return int64Size, f, nil

	case uint32:
		f := func(b []byte) N {
			return N(binary.LittleEndian.Uint32(b))
		}
		return uint32Size, f, nil

	case uint64:
		f := func(b []byte) N {
			return N(binary.LittleEndian.Uint64(b))
		}
		return uint64Size, f, nil

	case int:
		f := func(b []byte) N {
			return N(int(binary.LittleEndian.Uint64(b)))
		}
		return intSize, f, nil

	case uint:
		f := func(b []byte) N {
			return N(uint(binary.LittleEndian.Uint64(b)))
		}
		return uintSize, f, nil
	case int16:
		f := func(b []byte) N {
			return N(int16(binary.LittleEndian.Uint16(b)))
		}
		return int16Size, f, nil

	case uint16:
		f := func(b []byte) N {
			return N(binary.LittleEndian.Uint16(b))
		}
		return uint16Size, f, nil

	case int8:
		f := func(b []byte) N {
			return N(int8(b[0]))
		}
		return int8Size, f, nil

	case uint8:
		f := func(b []byte) N {
			return N(b[0])
		}
		return uint8Size, f, nil

	case uintptr:
		f := func(b []byte) N {
			return N(uintptr(binary.LittleEndian.Uint64(b)))
		}
		return uintptrSize, f, nil
	default:
		return 0, nil, fmt.Errorf("unsupported coordinate numeric type: %T", zero)
	}
}

// WriteTo serializes RTreeGN to an io.Writer using pre-order depth-first traversal.
func (tr *RTreeGN[N, T]) WriteTo(w io.Writer, writeItems func(w io.Writer, items []T) error) error {
	if tr.count > math.MaxInt32 {
		return fmt.Errorf("tree count %d exceeds maximum 32-bit limit %d", tr.count, math.MaxInt32)
	}
	if tr.count > 0 && tr.root == nil {
		return errors.New("corrupt tree: positive count with nil root")
	}

	eSize, typeId, put, err := getCoordEncoder[N]()
	if err != nil {
		return err
	}

	rectSize := eSize * 4
	buf := make([]byte, maxEntries*rectSize)

	writeRects := func(w io.Writer, rects []rect[N]) error {
		size := len(rects) * rectSize
		for i, r := range rects {
			off := i * rectSize
			put(buf[off:off+eSize], r.min[0])
			put(buf[off+eSize:off+eSize*2], r.min[1])
			put(buf[off+eSize*2:off+eSize*3], r.max[0])
			put(buf[off+eSize*3:off+eSize*4], r.max[1])
		}
		_, err := w.Write(buf[:size])
		return err
	}

	// write file header (16 bytes)
	copy(buf[0:4], magicNumber[:])
	binary.LittleEndian.PutUint16(buf[4:6], SoftwareVersionMajor)  // Major
	binary.LittleEndian.PutUint16(buf[6:8], SoftwareVersionMinor)  // Minor
	binary.LittleEndian.PutUint16(buf[8:10], SoftwareVersionPatch) // Patch
	buf[10] = typeId
	if tr.static {
		buf[11] = 0x01
	} else {
		buf[11] = 0x00
	}
	if _, err := w.Write(buf[0:12]); err != nil {
		return err
	}

	//  Write r-tree Metadata (5 bytes)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(tr.count))
	if _, err := w.Write(buf[0:4]); err != nil {
		return err
	}

	put(buf[0:eSize], tr.rect.min[0])
	put(buf[eSize:eSize*2], tr.rect.min[1])
	put(buf[eSize*2:eSize*3], tr.rect.max[0])
	put(buf[eSize*3:eSize*4], tr.rect.max[1])
	if _, err := w.Write(buf[0:rectSize]); err != nil {
		return err
	}

	// Write r-tree nodes
	rootRect, itemsWritten, err := tr.writeNode(w, tr.root, writeRects, writeItems, buf, 0)
	if err != nil {
		return err
	}
	if tr.rect != rootRect {
		return fmt.Errorf("corrupt tree: bounding box mismatch: tree has %v, root node has %v", tr.rect, rootRect)
	}
	if itemsWritten != tr.count {
		return fmt.Errorf("corrupt tree: count mismatch: tree has %d, leaf items contain %d", tr.count, itemsWritten)
	}
	return nil
}

// ReadFrom deserializes an RTreeGN from an io.Reader.
func (tr *RTreeGN[N, T]) ReadFrom(r io.Reader, readItems func(r io.Reader, items []T) error) error {
	if tr == nil {
		return errors.New("tree must not be nil")
	}
	if r == nil {
		return errors.New("reader must not be nil")
	}

	eSize, _, _, err := getCoordEncoder[N]()
	if err != nil {
		return err
	}

	rectSize := eSize * 4
	buf := make([]byte, maxEntries*rectSize)

	// read and validate file header
	if _, err := io.ReadFull(r, buf[0:12]); err != nil {
		if errors.Is(err, io.EOF) {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	if !slices.Equal(magicNumber[:], buf[0:4]) {
		return fmt.Errorf("invalid R-Tree magic number: %v", buf[0:4])
	}

	major := binary.LittleEndian.Uint16(buf[4:6])
	if major != SoftwareVersionMajor {
		return fmt.Errorf("unsupported major version: %d", major)
	}
	minor := binary.LittleEndian.Uint16(buf[6:8])
	if minor != SoftwareVersionMinor {
		return fmt.Errorf("unsupported minor version: %d", minor)
	}
	patch := binary.LittleEndian.Uint16(buf[8:10])
	if patch != SoftwareVersionPatch {
		return fmt.Errorf("unsupported patch version: %d", patch)
	}
	dSize, get, err := getCoordDecoder[N](buf[10])
	if err != nil {
		return err
	}
	if (buf[11] & 0xFE) != 0 {
		return fmt.Errorf("invalid header flags: unsupported flag bits 0x%02X", buf[11]&0xFE)
	}

	var newTr RTreeGN[N, T]
	newTr.static = (buf[11] & 0x01) != 0
	newTr.qpool = tr.qpool

	// read r-tree metadata
	if _, err := io.ReadFull(r, buf[0:4]); err != nil {
		if errors.Is(err, io.EOF) {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	metaCount := binary.LittleEndian.Uint32(buf[0:4])
	if metaCount > math.MaxInt32 {
		return fmt.Errorf("invalid tree count %d: exceeds max positive int32", metaCount)
	}
	newTr.count = int(metaCount)

	if readItems == nil {
		return errors.New("readItems must not be nil")
	}

	readRects := func(r io.Reader, rects []rect[N]) error {
		size := len(rects) * rectSize
		if _, err := io.ReadFull(r, buf[:size]); err != nil {
			if errors.Is(err, io.EOF) {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		for i := range rects {
			off := i * rectSize
			rects[i].min[0] = get(buf[off : off+dSize])
			rects[i].min[1] = get(buf[off+dSize : off+dSize*2])
			rects[i].max[0] = get(buf[off+dSize*2 : off+dSize*3])
			rects[i].max[1] = get(buf[off+dSize*3 : off+dSize*4])
		}
		return nil
	}

	if _, err := io.ReadFull(r, buf[:rectSize]); err != nil {
		if errors.Is(err, io.EOF) {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	newTr.rect.min[0] = get(buf[0:dSize])
	newTr.rect.min[1] = get(buf[dSize : dSize*2])
	newTr.rect.max[0] = get(buf[dSize*2 : dSize*3])
	newTr.rect.max[1] = get(buf[dSize*3 : dSize*4])

	// read r-tree nodes
	root, itemsCount, err := newTr.readNode(r, readRects, readItems, buf, 0)
	if err != nil {
		return err
	}
	if itemsCount != newTr.count {
		return fmt.Errorf("tree item count mismatch: header has %d, leaves contain %d", newTr.count, itemsCount)
	}
	if newTr.rect != root.rect() {
		return fmt.Errorf("tree bounding box mismatch: header has %v, root node has %v", newTr.rect, root.rect())
	}
	newTr.root = root
	*tr = newTr
	return nil
}

func (tr *RTreeGN[N, T]) writeNode(w io.Writer, n *node[N, T], writeRects func(w io.Writer, rects []rect[N]) error, writeItems func(w io.Writer, items []T) error, buf []byte, depth int) (rect[N], int, error) {

	if n == nil {
		return rect[N]{}, 0, errors.New("corrupt tree: node is nil")
	}
	if n.count <= 0 || n.count > maxEntries {
		return rect[N]{}, 0, fmt.Errorf("corrupt node: invalid entry count %d", n.count)
	}

	buf[0] = byte(n.kind)
	binary.LittleEndian.PutUint16(buf[1:3], uint16(n.count))
	if _, err := w.Write(buf[0:3]); err != nil {
		return rect[N]{}, 0, err
	}

	// Write active bounding boxes in bulk
	k := int(n.count)
	if err := writeRects(w, n.rects[:k]); err != nil {
		return rect[N]{}, 0, err
	}

	if n.leaf() {
		err := writeItems(w, n.items()[:k])
		if err != nil {
			return rect[N]{}, 0, err
		}
		// because this node n is leaf, return early
		return n.rect(), k, nil
	}

	// branch node
	// Recurse for all children
	children := n.children()
	if children == nil {
		return rect[N]{}, 0, fmt.Errorf("corrupt node: expected branch node but got kind %d", n.kind)
	}
	totalItems := 0
	for i := 0; i < k; i++ {
		if children[i] == nil {
			return rect[N]{}, 0, fmt.Errorf("corrupt node: branch child %d is nil", i)
		}
		childRect, subItems, err := tr.writeNode(w, children[i], writeRects, writeItems, buf, depth+1)
		if err != nil {
			return rect[N]{}, 0, err
		}
		if n.rects[i] != childRect {
			return rect[N]{}, 0, fmt.Errorf("corrupt node: branch child %d bounding box mismatch: node has %v, child has %v", i, n.rects[i], childRect)
		}
		totalItems += subItems
	}
	return n.rect(), totalItems, nil
}

func (tr *RTreeGN[N, T]) readNode(r io.Reader, readRects func(r io.Reader, rects []rect[N]) error, readItems func(r io.Reader, items []T) error, buf []byte, depth int) (*node[N, T], int, error) {

	if _, err := io.ReadFull(r, buf[:3]); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, 0, io.ErrUnexpectedEOF
		}
		return nil, 0, err
	}
	knd := kind(buf[0])
	cnt := binary.LittleEndian.Uint16(buf[1:3])
	if knd != leaf && knd != branch {
		return nil, 0, fmt.Errorf("invalid node kind: %d", buf[0])
	}
	if cnt == 0 || cnt > maxEntries {
		return nil, 0, fmt.Errorf("invalid node entry count: %d", cnt)
	}

	isLeaf := knd == leaf
	k := int(cnt)
	if k > tr.count {
		return nil, 0, fmt.Errorf("node entry count %d exceeded header count %d", k, tr.count)
	}
	n := tr.newNode(isLeaf)
	n.count = int16(cnt)

	if err := readRects(r, n.rects[:k]); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, 0, io.ErrUnexpectedEOF
		}
		return nil, 0, err
	}

	if isLeaf {
		// leaf node
		if readItems != nil {
			if err := readItems(r, n.items()[:k]); err != nil {
				if errors.Is(err, io.EOF) {
					return nil, 0, io.ErrUnexpectedEOF
				}
				return nil, 0, err
			}
		}
		return n, k, nil
	}

	// branch node: read children sequentially
	children := n.children()
	totalItems := 0
	for i := 0; i < k; i++ {
		child, subItems, err := tr.readNode(r, readRects, readItems, buf, depth+1)
		if err != nil {
			return nil, 0, err
		}
		if n.rects[i] != child.rect() {
			return nil, 0, fmt.Errorf("branch child %d bounding box mismatch: header has %v, child has %v", i, n.rects[i], child.rect())
		}
		children[i] = child
		totalItems += subItems
		if totalItems > tr.count {
			return nil, 0, fmt.Errorf("tree item count %d exceeded header count %d", totalItems, tr.count)
		}
	}
	return n, totalItems, nil
}
