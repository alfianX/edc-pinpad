package delivery

import (
	"edc-pinpad/infrastructure"
	"edc-pinpad/model"
	"edc-pinpad/usecase"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PinpadHandler struct {
	uc     *usecase.PinpadUsecase
	ws     *infrastructure.WSManager
	logger *slog.Logger
}

func NewPinpadHandler(uc *usecase.PinpadUsecase, ws *infrastructure.WSManager, logger *slog.Logger) *PinpadHandler {
	return &PinpadHandler{uc, ws, logger}
}

func (h *PinpadHandler) HandlePinCreate(c *gin.Context) {
	var req model.PinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	h.logger.Info("Received PIN Create Request", slog.String("transactionId", req.TransactionId))

	if err := h.uc.ProcessPIN(req); err != nil {
		h.logger.Error("Failed to process PIN create", slog.String("error", err.Error()))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Successfully processed PIN create", slog.String("transactionId", req.TransactionId))

	c.JSON(http.StatusOK, gin.H{"responseCode": "00", "status": "PINPAD_READY"})
}

func (h *PinpadHandler) HandlePinChange(c *gin.Context) {
	var req model.PinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	h.logger.Info("Received PIN Change Request", slog.String("transactionId", req.TransactionId))

	if err := h.uc.ProcessPIN(req); err != nil {
		h.logger.Error("Failed to process PIN change", slog.String("error", err.Error()))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Successfully processed PIN change", slog.String("transactionId", req.TransactionId))

	c.JSON(http.StatusOK, gin.H{"responseCode": "00", "status": "PINPAD_READY"})
}

func (h *PinpadHandler) HandlePinVerify(c *gin.Context) {
	var req model.PinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	h.logger.Info("Received PIN Verify Request", slog.String("transactionId", req.TransactionId))

	if err := h.uc.ProcessPIN(req); err != nil {
		h.logger.Error("Failed to process PIN verify", slog.String("error", err.Error()))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Successfully processed PIN verify", slog.String("transactionId", req.TransactionId))

	c.JSON(http.StatusOK, gin.H{"responseCode": "00", "status": "PINPAD_READY"})
}

func (h *PinpadHandler) HandlePenarikanTunai(c *gin.Context) {
	var req model.PenarikanTunaiReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	h.logger.Info("Received Penarikan Tunai Request", slog.String("transactionId", req.TransactionId))

	if err := h.uc.ProcessPenarikanTunai(req); err != nil {
		h.logger.Error("Failed to process Penarikan Tunai", slog.String("error", err.Error()))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Successfully processed Penarikan Tunai", slog.String("transactionId", req.TransactionId))

	c.JSON(http.StatusOK, gin.H{"responseCode": "00", "status": "PINPAD_READY"})
}

func (h *PinpadHandler) HandleTransfer(c *gin.Context) {
	var req model.TransferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	h.logger.Info("Received Transfer Request", slog.String("transactionId", req.TransactionId))

	if err := h.uc.ProcessTransfer(req); err != nil {
		h.logger.Error("Failed to process Transfer", slog.String("error", err.Error()))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Successfully processed Transfer", slog.String("transactionId", req.TransactionId))

	c.JSON(http.StatusOK, gin.H{"responseCode": "00", "status": "PINPAD_READY"})
}

func (h *PinpadHandler) HandleWS(c *gin.Context) {
	tid := c.GetHeader("X-TID")
	conn, err := h.ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	h.ws.RegisterConn(tid, conn)
	defer h.ws.UnregisterConn(tid)

	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		mti, ok := msg["mti"].(string)
		if !ok {
			h.logger.Error("Invalid or missing 'mti' in EDC response", slog.String("tid", tid))
			_ = h.ws.SendJSON(tid, map[string]interface{}{"transactionId": tid, "responseCode": "05", "responseMessage": "Invalid MTI in response"})
			return
		}

		if mti != "0800" {
			h.uc.ProcessEDCResponse(tid, msg)
		} else {
			h.uc.ProcessEDCLogon(tid, msg)
		}
	}
}
