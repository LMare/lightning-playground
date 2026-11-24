package integration

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	channelsPath = "/lightning/channels"
	closePath    = "/lightning/channel/close"
)

var (
	chTx1 = strings.Repeat("a1", 32)
	chTx2 = strings.Repeat("b2", 32)
	chTx3 = strings.Repeat("c3", 32)
)

// repeatedBytes builds a 32-byte txid made of one repeated byte (byte reversal is then the identity).
func hexNib(c byte) byte {
	if c >= 'a' {
		return c - 'a' + 10
	}
	return c - '0'
}

func repeatedBytes(h string) []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = hexNib(h[0])<<4 | hexNib(h[1])
	}
	return b
}

func mockChannels(h *testsupport.TestHarness) {
	h.GrpcFactory.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(&lnrpc.ListChannelsResponse{Channels: []*lnrpc.Channel{
		{ChannelPoint: chTx1 + ":0", RemotePubkey: "02aabbccddeeff0011223344", Capacity: 1000, LocalBalance: 600, RemoteBalance: 390, Active: true},
		{ChannelPoint: chTx2 + ":1", RemotePubkey: "03aabbccddeeff0011223344", Capacity: 2000, Active: false},
	}}, nil)
	h.GrpcFactory.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(&lnrpc.PendingChannelsResponse{
		PendingOpenChannels: []*lnrpc.PendingChannelsResponse_PendingOpenChannel{{Channel: &lnrpc.PendingChannelsResponse_PendingChannel{
			RemoteNodePub: "04aabbccddeeff0011223344", ChannelPoint: chTx3 + ":2", Capacity: 300,
		}}},
	}, nil)
}

func mockCloseOK(h *testsupport.TestHarness) {
	h.GrpcFactory.Lightning.On("CloseChannel", mock.Anything, mock.Anything).Return(&testsupport.CloseStream{Updates: []*lnrpc.CloseStatusUpdate{
		{Update: &lnrpc.CloseStatusUpdate_ClosePending{ClosePending: &lnrpc.PendingUpdate{Txid: repeatedBytes("ee")}}},
	}}, nil)
}

func assertNoClose(t *testing.T, h *testsupport.TestHarness) {
	h.GrpcFactory.Lightning.AssertNotCalled(t, "CloseChannel", mock.Anything, mock.Anything)
}

func TestChannels_List_JSON(t *testing.T) {
	h := testsupport.NewTestHarness()
	mockChannels(h)

	rr := doPlain(h, http.MethodGet, channelsPath+"?id=lnd-0", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	var got []map[string]any
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Len(t, got, 3)
	assert.Equal(t, chTx1+":0", got[0]["channelPoint"])
	assert.Equal(t, "active", got[0]["status"])
	assert.Equal(t, "Active", got[0]["statusLabel"])
	assert.Equal(t, true, got[0]["closable"])
	assert.EqualValues(t, 1000, got[0]["capacitySat"])
	assert.EqualValues(t, 600, got[0]["localBalanceSat"])
	assert.EqualValues(t, 390, got[0]["remoteBalanceSat"])
	assert.Equal(t, "02aabbccddeeff0011223344", got[0]["remotePubKey"])
	assert.Equal(t, "inactive", got[1]["status"])
	assert.Equal(t, true, got[1]["closable"])
	assert.Equal(t, "pending_open", got[2]["status"])
	assert.Equal(t, false, got[2]["closable"])
}

func TestChannels_List_JSON_Empty(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(&lnrpc.ListChannelsResponse{}, nil)
	h.GrpcFactory.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(&lnrpc.PendingChannelsResponse{}, nil)

	rr := doPlain(h, http.MethodGet, channelsPath+"?id=lnd-0", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, "[]", rr.Body.String())
}

func TestChannels_List_HTMX(t *testing.T) {
	h := testsupport.NewTestHarness()
	mockChannels(h)

	rr := doHTMX(h, http.MethodGet, channelsPath+"?id=lnd-0", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	body := rr.Body.String()
	for _, tx := range []string{chTx1 + ":0", chTx2 + ":1", chTx3 + ":2"} {
		assert.Contains(t, body, `data-channel-point="`+tx+`"`)
	}
	assert.Contains(t, body, `hx-post="/lightning/channel/close"`)
	assert.NotContains(t, body, "__API_PREFIX__")
	// close forms only for the two closable rows, not for the pending one
	assert.Equal(t, 4, strings.Count(body, `hx-post="/lightning/channel/close"`))
	assert.Equal(t, 2, strings.Count(body, `value="`+chTx1+`:0"`))
	assert.Equal(t, 2, strings.Count(body, `value="`+chTx2+`:1"`))
	assert.Equal(t, 0, strings.Count(body, `value="`+chTx3+`:2"`))
	assert.Contains(t, body, `name="force"`)
	assert.Contains(t, body, `name="channelPoint"`)
	assert.NotContains(t, body, "No channel")

	// peer column is a row header carrying the full pubkey as title (one per channel)
	assert.Equal(t, 3, strings.Count(body, `<th scope="row"`))
	for _, pk := range []string{"02aabbccddeeff0011223344", "03aabbccddeeff0011223344", "04aabbccddeeff0011223344"} {
		assert.Contains(t, body, `title="`+pk+`"`)
	}
	assert.NotContains(t, body, "<td title=")

	// confirmation texts are kept (one pair per closable row)
	assert.Equal(t, 2, strings.Count(body, `hx-confirm="Close this channel cooperatively?"`))
	assert.Equal(t, 2, strings.Count(body, `hx-confirm="Force close this channel? Your funds stay locked for the CSV delay."`))
	// labels used by the global confirm modal
	assert.Equal(t, 2, strings.Count(body, `data-confirm-label="Close channel"`))
	assert.Equal(t, 2, strings.Count(body, `data-confirm-label="Force close"`))
}

func TestChannels_List_HTMX_Empty(t *testing.T) {
	h := testsupport.NewTestHarness()
	h.GrpcFactory.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(&lnrpc.ListChannelsResponse{}, nil)
	h.GrpcFactory.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(&lnrpc.PendingChannelsResponse{}, nil)

	rr := doHTMX(h, http.MethodGet, channelsPath+"?id=lnd-0", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "No channel")
	assert.Contains(t, rr.Body.String(), `colspan="6"`)
	assert.NotContains(t, rr.Body.String(), `scope="row"`)
	assert.NotContains(t, rr.Body.String(), `hx-post="/lightning/channel/close"`)
}

func TestChannels_List_Errors(t *testing.T) {
	h := testsupport.NewTestHarness()
	assert.Equal(t, http.StatusBadRequest, doPlain(h, http.MethodGet, channelsPath, nil).Code)
	assert.Equal(t, http.StatusBadRequest, doPlain(h, http.MethodGet, channelsPath+"?id=../etc", nil).Code)

	h.GrpcFactory.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(nil, errors.New("lnd down"))
	h.GrpcFactory.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(&lnrpc.PendingChannelsResponse{}, nil).Maybe()
	assert.Equal(t, http.StatusInternalServerError, doPlain(h, http.MethodGet, channelsPath+"?id=lnd-0", nil).Code)

	rr := doHTMX(h, http.MethodGet, channelsPath+"?id=lnd-0", nil)
	assert.NotContains(t, rr.Body.String(), "data-channel-point")
	assert.NotEmpty(t, rr.Body.String())
}

func closeForm(point, force string) url.Values {
	f := url.Values{"id": {"lnd-0"}, "channelPoint": {point}}
	if force != "" {
		f.Set("force", force)
	}
	return f
}

func TestChannels_Close_Cooperative(t *testing.T) {
	h := testsupport.NewTestHarness()
	mockChannels(h)
	mockCloseOK(h)

	rr := doPlain(h, http.MethodPost, closePath, closeForm(chTx1+":0", ""))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `{"closingTxId":"`+strings.Repeat("ee", 32)+`"}`, rr.Body.String())
	// Non-HTMX (JSON) request: no HTMX response header.
	assert.Empty(t, rr.Header().Get("HX-Trigger"))
	h.GrpcFactory.Lightning.AssertCalled(t, "CloseChannel", mock.Anything, mock.MatchedBy(func(r *lnrpc.CloseChannelRequest) bool {
		return !r.Force && r.GetChannelPoint().GetFundingTxidStr() == chTx1 && r.GetChannelPoint().GetOutputIndex() == 0
	}))
}

func TestChannels_Close_ForceInactive_HTMX(t *testing.T) {
	h := testsupport.NewTestHarness()
	mockChannels(h)
	mockCloseOK(h)

	rr := doHTMX(h, http.MethodPost, closePath, closeForm(chTx2+":1", "true"))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "channel-closed", rr.Header().Get("HX-Trigger"))
	assert.Contains(t, rr.Body.String(), "Channel closing")
	h.GrpcFactory.Lightning.AssertCalled(t, "CloseChannel", mock.Anything, mock.MatchedBy(func(r *lnrpc.CloseChannelRequest) bool {
		return r.Force && r.GetChannelPoint().GetFundingTxidStr() == chTx2 && r.GetChannelPoint().GetOutputIndex() == 1
	}))
}

func TestChannels_Close_BusinessErrors(t *testing.T) {
	cases := []struct {
		name string
		form url.Values
	}{
		{"missing id", url.Values{"channelPoint": {chTx1 + ":0"}}},
		{"invalid id", url.Values{"id": {"../x"}, "channelPoint": {chTx1 + ":0"}}},
		{"missing point", url.Values{"id": {"lnd-0"}}},
		{"invalid point", closeForm("garbage", "")},
		{"invalid force", closeForm(chTx1+":0", "maybe")},
		{"unknown channel", closeForm(chTx1+":9", "")},
		{"pending channel", closeForm(chTx3+":2", "true")},
		{"cooperative on inactive", closeForm(chTx2+":1", "false")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := testsupport.NewTestHarness()
			mockChannels(h)
			mockCloseOK(h)

			rr := doPlain(h, http.MethodPost, closePath, c.form)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Empty(t, rr.Header().Get("HX-Trigger"))
			assertNoClose(t, h)
		})
	}
}

func TestChannels_Close_LndError(t *testing.T) {
	h := testsupport.NewTestHarness()
	mockChannels(h)
	h.GrpcFactory.Lightning.On("CloseChannel", mock.Anything, mock.Anything).Return(nil, errors.New("lnd down"))

	rr := doPlain(h, http.MethodPost, closePath, closeForm(chTx1+":0", ""))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Empty(t, rr.Header().Get("HX-Trigger"))
}

func TestChannels_Close_MethodNotAllowed(t *testing.T) {
	h := testsupport.NewTestHarness()
	rr := doPlain(h, http.MethodGet, closePath+"?id=lnd-0", nil)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}
