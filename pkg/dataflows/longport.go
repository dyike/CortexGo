package dataflows

import (
	"context"
	"errors"

	lpconfig "github.com/longportapp/openapi-go/config"
	"github.com/longportapp/openapi-go/quote"
	"github.com/longportapp/openapi-go/trade"
)

type LongportConfig struct {
	AppKey      string
	AppSecret   string
	AccessToken string
}

type LongportClient struct {
	tradeCtx *trade.TradeContext
	quoteCtx *quote.QuoteContext
}

func NewLongportClient(cfg LongportConfig) (*LongportClient, error) {
	if cfg.AppKey == "" || cfg.AppSecret == "" || cfg.AccessToken == "" {
		return nil, errors.New("longport API credentials not configured")
	}

	conf, err := lpconfig.New(lpconfig.WithConfigKey(cfg.AppKey, cfg.AppSecret, cfg.AccessToken))
	if err != nil {
		return nil, err
	}

	tradeContext, err := trade.NewFromCfg(conf)
	if err != nil {
		return nil, err
	}

	quoteContext, err := quote.NewFromCfg(conf)
	if err != nil {
		return nil, err
	}

	return &LongportClient{
		tradeCtx: tradeContext,
		quoteCtx: quoteContext,
	}, nil
}

func (lpc *LongportClient) GetStaticInfo(ctx context.Context, symbols []string) (staticInfos []*quote.StaticInfo, err error) {
	if lpc.quoteCtx != nil {
		return lpc.quoteCtx.StaticInfo(ctx, symbols)
	}
	return nil, errors.New("quote context is nil")
}

func (lpc *LongportClient) GetSticksWithDay(ctx context.Context, symbols string, count int) (sticks []*quote.Candlestick, err error) {
	if lpc.quoteCtx != nil {
		return lpc.quoteCtx.Candlesticks(ctx, symbols, quote.PeriodDay, int32(count), quote.AdjustTypeNo)
	}
	return nil, errors.New("trade context is nil")
}
