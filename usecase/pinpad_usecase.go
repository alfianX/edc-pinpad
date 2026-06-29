package usecase

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"edc-pinpad/config"
	"edc-pinpad/infrastructure"
	"edc-pinpad/model"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type PinpadUsecase struct {
	db              *gorm.DB
	ws              *infrastructure.WSManager
	logger          *slog.Logger
	pendingRequests sync.Map
}

func NewPinpadUsecase(db *gorm.DB, ws *infrastructure.WSManager, l *slog.Logger) *PinpadUsecase {
	return &PinpadUsecase{db: db, ws: ws, logger: l}
}

func (u *PinpadUsecase) ProcessPIN(req model.PinReq) error {
	tx := model.Transaction{
		TransactionId:          req.TransactionId,
		PrimaryAccountNumber:   req.PrimaryAccountNumber,
		ProcessingCode:         req.ProcessingCode,
		BranchId:               req.BranchId,
		InitiatorId:            req.InitiatorId,
		TransmissionDateTime:   req.TransmissionDateTime,
		CardAcceptorTerminalIp: req.CardAcceptorTerminalIp,
		CardAcceptorTerminalId: req.CardAcceptorTerminalId,
		SourceAccountNumber:    req.SourceAccountNumber,
		SourceAccountName:      req.SourceAccountName,
		Status:                 "WAITING_EDC",
	}
	err := u.db.Create(&tx)
	if err.Error != nil {
		u.logger.Error("Failed to create transaction record", slog.String("tx", req.TransactionId), slog.Any("error", err.Error))
		return err.Error
	}
	return u.ws.SendJSON(req.CardAcceptorTerminalId, req)
}

func (u *PinpadUsecase) ProcessData(req model.PinReq) (map[string]interface{}, error) {
	tx := model.Transaction{
		TransactionId:          req.TransactionId,
		ProcessingCode:         req.ProcessingCode,
		BranchId:               req.BranchId,
		InitiatorId:            req.InitiatorId,
		TransmissionDateTime:   req.TransmissionDateTime,
		CardAcceptorTerminalIp: req.CardAcceptorTerminalIp,
		CardAcceptorTerminalId: req.CardAcceptorTerminalId,
		SourceAccountNumber:    req.SourceAccountNumber,
		SourceAccountName:      req.SourceAccountName,
		Status:                 "WAITING_EDC",
	}
	err := u.db.Create(&tx)
	if err.Error != nil {
		u.logger.Error("Failed to create transaction record", slog.String("tx", req.TransactionId), slog.Any("error", err.Error))
		return nil, err.Error
	}

	ch := make(chan map[string]interface{}, 1)
	u.pendingRequests.Store(req.TransactionId, ch)
	defer u.pendingRequests.Delete(req.TransactionId)

	if err := u.ws.SendJSON(req.CardAcceptorTerminalId, req); err != nil {
		return nil, err
	}

	select {
	case edcData := <-ch:
		u.logger.Info("Resumed Process Get Data from EDC response", slog.String("tx", req.TransactionId))

		// --- LOGIC DARI ProcessEDCResponse SEBELUMNYA DIPINDAH KESINI ---

		// Validasi MTI & Procode
		if edcData["mti"] != "0600" || edcData["processingCode"] != "950000" {
			u.ws.SendJSON(req.CardAcceptorTerminalId, map[string]interface{}{"transactionId": req.TransactionId, "responseCode": "05", "responseMessage": "Invalid MTI/Procode"})
			return nil, errors.New("invalid MTI or Procode from EDC")
		}

		// Update DB ke COMPLETED
		u.db.Model(&model.Transaction{}).Where("transaction_id = ?", tx.TransactionId).Updates(map[string]interface{}{
			"status": "COMPLETED",
			// ... update field response host lainnya ...
		})

		// Bales ke depan (Frontend/HTTP)
		return edcData, nil

	case <-time.After(120 * time.Second): // Timeout 120 detik
		// Update DB jika timeout
		u.db.Model(&model.Transaction{}).Where("transaction_id = ?", tx.TransactionId).Update("status", "TIMEOUT")
		u.logger.Error("Timeout waiting for EDC response", slog.String("tx", req.TransactionId))
		return nil, errors.New("edc timeout")
	}
}

func (u *PinpadUsecase) ProcessFace(req model.PinReq) (map[string]interface{}, error) {
	tx := model.Transaction{
		TransactionId:          req.TransactionId,
		ProcessingCode:         req.ProcessingCode,
		BranchId:               req.BranchId,
		InitiatorId:            req.InitiatorId,
		TransmissionDateTime:   req.TransmissionDateTime,
		CardAcceptorTerminalIp: req.CardAcceptorTerminalIp,
		CardAcceptorTerminalId: req.CardAcceptorTerminalId,
		SourceAccountNumber:    req.SourceAccountNumber,
		SourceAccountName:      req.SourceAccountName,
		Status:                 "WAITING_EDC",
	}
	err := u.db.Create(&tx)
	if err.Error != nil {
		u.logger.Error("Failed to create transaction record", slog.String("tx", req.TransactionId), slog.Any("error", err.Error))
		return nil, err.Error
	}

	ch := make(chan map[string]interface{}, 1)
	u.pendingRequests.Store(req.TransactionId, ch)
	defer u.pendingRequests.Delete(req.TransactionId)

	if err := u.ws.SendJSON(req.CardAcceptorTerminalId, req); err != nil {
		return nil, err
	}

	select {
	case edcData := <-ch:
		u.logger.Info("Resumed Process Face from EDC response", slog.String("tx", req.TransactionId))

		// --- LOGIC DARI ProcessEDCResponse SEBELUMNYA DIPINDAH KESINI ---

		// Validasi MTI & Procode
		if edcData["mti"] != "0600" || edcData["processingCode"] != "960000" {
			u.ws.SendJSON(req.CardAcceptorTerminalId, map[string]interface{}{"transactionId": req.TransactionId, "responseCode": "05", "responseMessage": "Invalid MTI/Procode"})
			return nil, errors.New("invalid MTI or Procode from EDC")
		}

		// Update DB ke COMPLETED
		u.db.Model(&model.Transaction{}).Where("transaction_id = ?", tx.TransactionId).Updates(map[string]interface{}{
			"status": "COMPLETED",
			// ... update field response host lainnya ...
		})

		// Bales ke depan (Frontend/HTTP)
		return edcData, nil

	case <-time.After(120 * time.Second): // Timeout 120 detik
		// Update DB jika timeout
		u.db.Model(&model.Transaction{}).Where("transaction_id = ?", tx.TransactionId).Update("status", "TIMEOUT")
		u.logger.Error("Timeout waiting for EDC response", slog.String("tx", req.TransactionId))
		return nil, errors.New("edc timeout")
	}
}

func (u *PinpadUsecase) ProcessPenarikanTunai(req model.PenarikanTunaiReq) error {
	tx := model.Transaction{
		TransactionId:          req.TransactionId,
		PrimaryAccountNumber:   req.PrimaryAccountNumber,
		ProcessingCode:         req.ProcessingCode,
		BranchId:               req.BranchId,
		InitiatorId:            req.InitiatorId,
		TransmissionDateTime:   req.TransmissionDateTime,
		CardAcceptorTerminalIp: req.CardAcceptorTerminalIp,
		CardAcceptorTerminalId: req.CardAcceptorTerminalId,
		SourceAccountNumber:    req.SourceAccountNumber,
		SourceAccountName:      req.SourceAccountName,
		TransactionFee:         req.TransactionFee,
		TransactionAmount:      req.TransactionAmount,
		Currency:               req.Currency,
		Status:                 "WAITING_EDC",
	}
	err := u.db.Create(&tx)
	if err.Error != nil {
		u.logger.Error("Failed to create transaction record", slog.String("tx", req.TransactionId), slog.Any("error", err.Error))
		return err.Error
	}
	return u.ws.SendJSON(req.CardAcceptorTerminalId, req)
}

func (u *PinpadUsecase) ProcessTransfer(req model.TransferReq) error {
	tx := model.Transaction{
		TransactionId:            req.TransactionId,
		PrimaryAccountNumber:     req.PrimaryAccountNumber,
		ProcessingCode:           req.ProcessingCode,
		BranchId:                 req.BranchId,
		InitiatorId:              req.InitiatorId,
		TransmissionDateTime:     req.TransmissionDateTime,
		CardAcceptorTerminalIp:   req.CardAcceptorTerminalIp,
		CardAcceptorTerminalId:   req.CardAcceptorTerminalId,
		SourceAccountNumber:      req.SourceAccountNumber,
		SourceAccountName:        req.SourceAccountName,
		DestinationAccountNumber: req.DestinationAccountNumber,
		DestinationAccountName:   req.DestinationAccountName,
		TransactionFee:           req.TransactionFee,
		TransactionAmount:        req.TransactionAmount,
		Currency:                 req.Currency,
		Status:                   "WAITING_EDC",
	}
	err := u.db.Create(&tx)
	if err.Error != nil {
		u.logger.Error("Failed to create transaction record", slog.String("tx", req.TransactionId), slog.Any("error", err.Error))
		return err.Error
	}
	return u.ws.SendJSON(req.CardAcceptorTerminalId, req)
}

func (u *PinpadUsecase) ProcessEDCResponse(tid string, data map[string]interface{}) {
	u.logger.Info("Received message from EDC", slog.String("tid", tid), slog.Any("data", data))

	txID, ok := data["transactionId"].(string)
	if !ok {
		u.logger.Error("Invalid or missing 'transactionId' in EDC response", slog.String("tid", tid))
		return
	}

	ch, exists := u.pendingRequests.Load(txID)
	if exists {
		// Jika ada, lempar data mentah dari EDC ke channel tersebut.
		// Proses validasi dan routing host akan dilanjutkan oleh fungsi yang menunggu.
		ch.(chan map[string]interface{}) <- data
		return
	}

	mti, ok := data["mti"].(string)
	if !ok {
		u.logger.Error("Invalid or missing 'mti' in EDC response", slog.String("tx", txID))
		_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Invalid MTI in response"})
		return
	}
	procode, ok := data["processingCode"].(string)
	if !ok {
		u.logger.Error("Invalid or missing 'processingCode' in EDC response", slog.String("tx", txID))
		_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Invalid Processing Code in response"})
	}

	var dataTrx model.Transaction
	err := u.db.Model(&model.Transaction{}).Where("transaction_id = ?", txID).First(&dataTrx).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			u.logger.Warn("Transaction not found for EDC response", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Transaction not found"})
		} else {
			u.logger.Error("Database error while fetching transaction", slog.String("tx", txID), slog.Any("error", err.Error))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Internal server error"})
		}
		return
	}

	if dataTrx.Status != "WAITING_EDC" {
		u.logger.Warn("Received EDC response for transaction not in WAITING_EDC status", slog.String("tx", txID), slog.String("currentStatus", dataTrx.Status))
		_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Invalid transaction status"})
		return
	}

	switch {
	case mti == "0600" && procode == "930000":
		u.logger.Info("Processing PIN create response", slog.String("tx", txID))
		if data["primaryAccountNumber"] != dataTrx.PrimaryAccountNumber {
			u.logger.Warn("Primary account number mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Primary account number mismatch"})
			return
		}
	case mti == "0600" && procode == "940000":
		u.logger.Info("Processing PIN change response", slog.String("tx", txID))
		if data["primaryAccountNumber"] != dataTrx.PrimaryAccountNumber {
			u.logger.Warn("Primary account number mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Primary account number mismatch"})
			return
		}
	case mti == "0600" && procode == "970000":
		u.logger.Info("Processing Re-issue PIN response", slog.String("tx", txID))
		if data["primaryAccountNumber"] != dataTrx.PrimaryAccountNumber {
			u.logger.Warn("Primary account number mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Primary account number mismatch"})
			return
		}
	case mti == "0100" && procode == "810000":
		u.logger.Info("Processing PIN verify response", slog.String("tx", txID))
		if data["primaryAccountNumber"] != dataTrx.PrimaryAccountNumber {
			u.logger.Warn("Primary account number mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Primary account number mismatch"})
			return
		}
	case mti == "0100" && procode == "010000":
		u.logger.Info("Processing Penarikan Tunai response", slog.String("tx", txID))
		if data["primaryAccountNumber"] != dataTrx.PrimaryAccountNumber {
			u.logger.Warn("Primary account number mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Primary account number mismatch"})
			return
		}

		if data["sourceAccountNumber"] != dataTrx.SourceAccountNumber {
			u.logger.Warn("Source account number mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Source account number mismatch"})
			return
		}

		if data["transactionFee"] != dataTrx.TransactionFee {
			u.logger.Warn("Transaction fee mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Transaction fee mismatch"})
			return
		}

		if data["transactionAmount"] != dataTrx.TransactionAmount {
			u.logger.Warn("Transaction amount mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Transaction amount mismatch"})
			return
		}

		if data["currency"] != dataTrx.Currency {
			u.logger.Warn("Currency mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Currency mismatch"})
			return
		}
	case mti == "0100" && procode == "400000":
		u.logger.Info("Processing Transfer response", slog.String("tx", txID))
		if data["primaryAccountNumber"] != dataTrx.PrimaryAccountNumber {
			u.logger.Warn("Primary account number mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Primary account number mismatch"})
			return
		}

		if data["sourceAccountNumber"] != dataTrx.SourceAccountNumber {
			u.logger.Warn("Source account number mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Source account number mismatch"})
			return
		}

		if data["sourceAccountName"] != dataTrx.SourceAccountName {
			u.logger.Warn("Source account name mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Source account name mismatch"})
			return
		}

		if data["destinationAccountNumber"] != dataTrx.DestinationAccountNumber {
			u.logger.Warn("Destination account number mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Destination account number mismatch"})
			return
		}

		if data["destinationAccountName"] != dataTrx.DestinationAccountName {
			u.logger.Warn("Destination account name mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Destination account name mismatch"})
			return
		}

		if data["transactionFee"] != dataTrx.TransactionFee {
			u.logger.Warn("Transaction fee mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Transaction fee mismatch"})
			return
		}

		if data["transactionAmount"] != dataTrx.TransactionAmount {
			u.logger.Warn("Transaction amount mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Transaction amount mismatch"})
			return
		}

		if data["currency"] != dataTrx.Currency {
			u.logger.Warn("Currency mismatch", slog.String("tx", txID))
			_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Currency mismatch"})
			return
		}
	default:
		u.logger.Error("Unsupported transaction routing", slog.String("mti", mti), slog.String("procode", procode))
		_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Unsupported transaction routing"})
		return
	}

	errUpdt := u.db.Model(&model.Transaction{}).Where("transaction_id = ?", dataTrx.TransactionId).Updates(map[string]interface{}{
		"mti":                       data["mti"],
		"system_trace_audit_number": data["systemTraceAuditNumber"],
		"card_acceptor_merchant_id": data["cardAcceptorMerchantId"],
		"status":                    "FORWARDING_TO_HOST",
	}).Error
	if errUpdt != nil {
		u.logger.Error("Failed to update transaction record", slog.String("tx", txID), slog.Any("error", errUpdt))
		return
	}

	u.logger.Info("Forwarding transaction to host", slog.String("tx", txID))
	hostResp, err := u.callExternalHost(data)
	if err != nil {
		errUpdt := u.db.Model(&model.Transaction{}).Where("transaction_id = ?", dataTrx.TransactionId).Update("status", "HOST_UNREACHABLE").Error
		if errUpdt != nil {
			u.logger.Error("Failed to update transaction status to HOST_UNREACHABLE", slog.String("tx", txID), slog.Any("error", errUpdt))
		}
		u.logger.Error("Host unreachable", slog.String("tx", txID))
		_ = u.ws.SendJSON(tid, map[string]interface{}{"transactionId": txID, "responseCode": "05", "responseMessage": "Host unreachable"})
		return
	}

	u.db.Model(&model.Transaction{}).Where("transaction_id = ?", txID).Updates(map[string]interface{}{
		"response_code":              hostResp["responseCode"],
		"response_message":           hostResp["responseMessage"],
		"approval_code":              hostResp["approvalCode"],
		"retrieval_reference_number": hostResp["retrievalReferenceNumber"],
		"status":                     "COMPLETED",
	})

	u.logger.Info("Transaction completed and response to EDC", slog.String("tx", txID))
	_ = u.ws.SendJSON(tid, hostResp)
}

func (u *PinpadUsecase) ProcessEDCLogon(tid string, data map[string]interface{}) {
	u.logger.Info("Received logon from EDC", slog.String("tid", tid), slog.Any("data", data))

	_, ok := data["processingCode"].(string)
	if !ok {
		u.logger.Error("Invalid or missing 'processingCode' from EDC ", slog.String("tid", tid))
		return
	}

	_, ok = data["transmissionDateTime"].(string)
	if !ok {
		u.logger.Error("Invalid or missing 'transmissionDateTime' from EDC ", slog.String("tid", tid))
		return
	}

	_, ok = data["transmissionDateTime"].(string)
	if !ok {
		u.logger.Error("Invalid or missing 'transmissionDateTime' from EDC ", slog.String("tid", tid))
		return
	}

	_, ok = data["cardAcceptorMerchantId"].(string)
	if !ok {
		u.logger.Error("Invalid or missing 'cardAcceptorMerchantId' from EDC ", slog.String("tid", tid))
		return
	}

	_, ok = data["cardAcceptorTerminalId"].(string)
	if !ok {
		u.logger.Error("Invalid or missing 'cardAcceptorTerminalId' from EDC ", slog.String("tid", tid))
		return
	}

	u.logger.Info("Forwarding logon to host", slog.String("tid", tid))
	hostResp, err := u.callExternalHost(data)
	if err != nil {
		u.logger.Error("Host unreachable", slog.String("tid", tid))
		_ = u.ws.SendJSON(tid, map[string]interface{}{"responseCode": "05", "responseMessage": "Host unreachable"})
		return
	}

	u.logger.Info("Logon completed and response to EDC", slog.String("tid", tid), slog.Any("data", hostResp))
	_ = u.ws.SendJSON(tid, hostResp)
}

func (u *PinpadUsecase) callExternalHost(payload map[string]interface{}) (map[string]interface{}, error) {

	tid, ok := payload["cardAcceptorTerminalId"].(string)
	if !ok {
		return nil, errors.New("invalid or missing 'tid' in payload")
	}

	// procode, ok := payload["processingCode"].(string)
	// if !ok {
	// 	return nil, errors.New("invalid or missing 'processingCode' in payload")
	// }

	// 2. Tentukan Endpoint Path berdasarkan MTI & Procode
	var endpointPath string
	endpointPath = "/api/v1"
	// switch {
	// case mti == "0600" && procode == "930000":
	// 	endpointPath = "/api/pin/create"
	// case mti == "0600" && procode == "940000":
	// 	endpointPath = "/api/pin/change"
	// case mti == "0600" && procode == "970000":
	// 	endpointPath = "/api/pin/re_issue"
	// case mti == "0100" && procode == "810000":
	// 	endpointPath = "/api/pin/verify"
	// case mti == "0100" && procode == "010000":
	// 	endpointPath = "/api/cash/withdrawal"
	// case mti == "0100" && procode == "400000":
	// 	endpointPath = "/api/transfer/intrabank"
	// case mti == "0800" && procode == "9B0000":
	// 	endpointPath = "/api/logon"
	// default:
	// 	// Tolak request di awal jika transaksi tidak didukung
	// 	return nil, fmt.Errorf("unsupported transaction routing for MTI %s and Procode %s", mti, procode)
	// }

	// 3. Gabungkan Base URL dengan Endpoint Path
	config.GlobalConfig.Mu.RLock()
	baseURL := config.GlobalConfig.HostURL
	config.GlobalConfig.Mu.RUnlock()
	// TrimSuffix memastikan tidak ada double slash (//) jika HOST_URL diakhiri dengan slash
	baseURL = strings.TrimSuffix(baseURL, "/")
	fullURL := baseURL + endpointPath

	// 4. Proses Request (Existing Logic)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		// Fallback jika OS tidak punya data timezone (jarang terjadi)
		loc = time.FixedZone("WIB", 7*3600)
	}

	// 2. Ambil waktu saat ini dan paksa masuk ke zona waktu tersebut
	now := time.Now().In(loc)

	// 3. Format menggunakan RFC3339
	timestamp := now.Format(time.RFC3339)

	// 3. Generate HMAC-SHA512 Signature (Sesuai logic di Middleware Master)
	mac := hmac.New(sha512.New, []byte(config.GlobalConfig.ClientSecret))
	mac.Write([]byte(tid + timestamp))
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-timestamp", timestamp)
	req.Header.Set("x-partner-id", config.GlobalConfig.PartnerID)
	req.Header.Set("x-signature", signature)

	// u.logger.Info("Header ", slog.String("timestamp", timestamp), slog.String("partner id", config.GlobalConfig.PartnerID), slog.String("signature", signature))
	u.logger.Info("Request", slog.Any("Header", req.Header))

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
