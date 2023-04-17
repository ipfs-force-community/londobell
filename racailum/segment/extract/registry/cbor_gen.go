package registry

import (
	"fmt"
	"io"
	"math"
	"sort"

	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
)

var _ = fmt.Errorf
var _ = cid.Undef
var _ = math.E
var _ = sort.Sort

var lengthBufInputData = []byte{130}

func (t *InputData) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write(lengthBufInputData); err != nil {
		return err
	}

	// t.Function (string) (string)
	if len(t.Function) > cbg.MaxLength {
		return fmt.Errorf("Value in field t.Function was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Function))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, t.Function); err != nil {
		return err
	}

	// t.Params ([]*main.ConstractParams) (slice)
	if len(t.Params) > cbg.MaxLength {
		return fmt.Errorf("Slice value in field t.Params was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajArray, uint64(len(t.Params))); err != nil {
		return err
	}
	for _, v := range t.Params {
		if err := v.MarshalCBOR(cw); err != nil {
			return err
		}
	}
	return nil
}

func (t *InputData) UnmarshalCBOR(r io.Reader) (err error) {
	*t = InputData{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 2 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Function (string) (string)

	{
		sval, err := cbg.ReadString(cr)
		if err != nil {
			return err
		}

		t.Function = sval
	}
	// t.Params ([]*main.ConstractParams) (slice)

	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("t.Params: array too large (%d)", extra)
	}

	if maj != cbg.MajArray {
		return fmt.Errorf("expected cbor array")
	}

	if extra > 0 {
		t.Params = make([]*ConstractParams, extra)
	}

	for i := 0; i < int(extra); i++ {

		var v ConstractParams
		if err := v.UnmarshalCBOR(cr); err != nil {
			return err
		}

		t.Params[i] = &v
	}

	return nil
}

var lengthBufConstractParams = []byte{131}

func (t *ConstractParams) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write(lengthBufConstractParams); err != nil {
		return err
	}

	// t.Name (string) (string)
	if len(t.Name) > cbg.MaxLength {
		return fmt.Errorf("Value in field t.Name was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Name))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, t.Name); err != nil {
		return err
	}

	// t.Type (string) (string)
	if len(t.Type) > cbg.MaxLength {
		return fmt.Errorf("Value in field t.Type was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Type))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, t.Type); err != nil {
		return err
	}

	// t.Data (string) (string)
	if len(t.Data) > cbg.MaxLength {
		return fmt.Errorf("Value in field t.Data was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Data))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, t.Data); err != nil {
		return err
	}
	return nil
}

func (t *ConstractParams) UnmarshalCBOR(r io.Reader) (err error) {
	*t = ConstractParams{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 3 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Name (string) (string)

	{
		sval, err := cbg.ReadString(cr)
		if err != nil {
			return err
		}

		t.Name = sval
	}
	// t.Type (string) (string)

	{
		sval, err := cbg.ReadString(cr)
		if err != nil {
			return err
		}

		t.Type = sval
	}
	// t.Data (string) (string)

	{
		sval, err := cbg.ReadString(cr)
		if err != nil {
			return err
		}

		t.Data = sval
	}
	return nil
}
