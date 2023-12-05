package segment

import (
	"context"
	"fmt"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ipfs-force-community/londobell/lib/mgoutil"

	"github.com/ipfs-force-community/londobell/common"
)

// Segment store aggregate states of multi segments for db
type Segment struct {
	name string

	db     common.DocumentDB
	rdb    common.DocumentDB
	rcli   *mongo.Client
	wcli   *mongo.Client
	rdsn   string
	wdsn   string
	dbName string
	segMgr *SegManager
	opts   common2.Options
}

func New(ctx context.Context, segMgr *SegManager, config common2.Config) (*Segment, error) {
	activeSegName, has, err := segMgr.LoadActive()
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, fmt.Errorf("no active segment")
	}

	info, has, err := segMgr.LoadInfo(activeSegName)
	if err != nil {
		return nil, fmt.Errorf("load info: %w", err)
	}

	if !has {
		return nil, fmt.Errorf("info not found")
	}

	var (
		//multiWdocs = &mgoutil.MultiDB{}
		//rdoc common.DocumentDB

		wdoc, rdoc common.DocumentDB
		rcli, wcli *mongo.Client
	)

	//for _, write := range info.Writes {
	//	wcli, err := mgoutil.Connect(ctx, write)
	//	if err != nil {
	//		return nil, fmt.Errorf("connect to write db: %w", err)
	//	}
	//
	//	wdoc, err := mgoutil.NewMgoDocDB(ctx, wcli, wcli.Database(name))
	//	if err != nil {
	//		return nil, fmt.Errorf("construct write doc db: %w", err)
	//	}
	//
	//	err = multiWdocs.SetDbs(wdoc)
	//	if err != nil {
	//		return nil, fmt.Errorf("multiwdocs setdbs: %w", err)
	//	}
	//}

	if info.Read != "" {
		rcli, err = mgoutil.Connect(ctx, info.Read)
		if err != nil {
			return nil, fmt.Errorf("connect to read db: %w", err)
		}

		rdoc, err = mgoutil.NewMgoDocDB(ctx, rcli, rcli.Database(activeSegName))
		if err != nil {
			return nil, fmt.Errorf("construct read doc db: %w", err)
		}
	}

	if info.Write != "" {
		wcli, err = mgoutil.Connect(ctx, info.Write)
		if err != nil {
			return nil, fmt.Errorf("connect to write db: %w", err)
		}

		wdoc, err = mgoutil.NewMgoDocDB(ctx, wcli, wcli.Database(activeSegName))
		if err != nil {
			return nil, fmt.Errorf("construct write doc db: %w", err)
		}
	}

	return &Segment{
		name:   activeSegName,
		rcli:   rcli,
		wcli:   wcli,
		rdsn:   info.Read,
		wdsn:   info.Write,
		dbName: activeSegName,
		db:     wdoc,
		rdb:    rdoc,
		segMgr: segMgr,
		opts:   common2.NewOptions(config.BatchSegmentInsertLimit),
	}, nil
}

func (s *Segment) ReConnect(ctx context.Context) error {
	var err error
	if s.rdsn != "" {
		s.rcli, err = mgoutil.Connect(ctx, s.rdsn)
		if err != nil {
			return fmt.Errorf("connect to read db: %w", err)
		}

		s.rdb, err = mgoutil.NewMgoDocDB(ctx, s.rcli, s.rcli.Database(s.dbName))
		if err != nil {
			return fmt.Errorf("construct read doc db: %w", err)
		}
	}

	if s.wdsn != "" {
		s.wcli, err = mgoutil.Connect(ctx, s.wdsn)
		if err != nil {
			return fmt.Errorf("connect to write db: %w", err)
		}

		s.db, err = mgoutil.NewMgoDocDB(ctx, s.wcli, s.wcli.Database(s.dbName))
		if err != nil {
			return fmt.Errorf("construct write doc db: %w", err)
		}
	}
	return nil
}
