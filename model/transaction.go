package model

import (
	"time"

	"gorm.io/gorm"
)

// Transaction adalah tabel monitoring di MySQL
type Transaction struct {
	ID                       uint   `gorm:"primaryKey;autoIncrement"`
	TransactionId            string `gorm:"type:varchar(50);uniqueIndex"`
	Mti                      string `gorm:"type:varchar(4)"`
	PrimaryAccountNumber     string `gorm:"type:varchar(20)"`
	ProcessingCode           string `gorm:"type:varchar(10)"`
	TransmissionDateTime     string `gorm:"type:varchar(20)"`
	SystemTraceAuditNumber   string `gorm:"type:varchar(6)"`
	BranchId                 string `gorm:"type:varchar(10)"`
	InitiatorId              string `gorm:"type:varchar(20)"`
	CardAcceptorMerchantId   string `gorm:"type:varchar(50)"`
	CardAcceptorTerminalId   string `gorm:"type:varchar(20)"`
	CardAcceptorTerminalIp   string `gorm:"type:varchar(20)"`
	SourceAccountNumber      string `gorm:"type:varchar(20)"`
	SourceAccountName        string `gorm:"type:varchar(100)"`
	DestinationAccountNumber string `gorm:"type:varchar(20)"`
	DestinationAccountName   string `gorm:"type:varchar(100)"`
	TransactionFee           string `gorm:"type:varchar(20)"`
	TransactionAmount        string `gorm:"type:varchar(20)"`
	Currency                 string `gorm:"type:varchar(10)"`
	ResponseCode             string `gorm:"type:varchar(5)"`
	ResponseMessage          string `gorm:"type:varchar(150)"`
	RetrievalReferenceNumber string `gorm:"type:varchar(12)"`
	ApprovalCode             string `gorm:"type:varchar(10)"`
	Status                   string `gorm:"type:varchar(100)"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
	DeletedAt                gorm.DeletedAt `gorm:"index"`
}

// Request dari Master Gateway
type PinReq struct {
	PrimaryAccountNumber   string `json:"primaryAccountNumber"`
	SourceAccountNumber    string `json:"sourceAccountNumber"`
	SourceAccountName      string `json:"sourceAccountName"`
	ProcessingCode         string `json:"processingCode"`
	TransactionId          string `json:"transactionId"`
	BranchId               string `json:"branchId"`
	InitiatorId            string `json:"initiatorId"`
	TransmissionDateTime   string `json:"transmissionDateTime"`
	CardAcceptorTerminalIp string `json:"cardAcceptorTerminalIp"`
	CardAcceptorTerminalId string `json:"cardAcceptorTerminalId"`
}

// Request dari Master Gateway
type PenarikanTunaiReq struct {
	PrimaryAccountNumber   string `json:"primaryAccountNumber"`
	ProcessingCode         string `json:"processingCode"`
	TransactionId          string `json:"transactionId"`
	BranchId               string `json:"branchId"`
	InitiatorId            string `json:"initiatorId"`
	TransmissionDateTime   string `json:"transmissionDateTime"`
	CardAcceptorTerminalIp string `json:"cardAcceptorTerminalIp"`
	CardAcceptorTerminalId string `json:"cardAcceptorTerminalId"`
	SourceAccountNumber    string `json:"sourceAccountNumber"`
	SourceAccountName      string `json:"sourceAccountName"`
	TransactionFee         string `json:"transactionFee"`
	TransactionAmount      string `json:"transactionAmount"`
	Currency               string `json:"currency"`
}

// Request dari Master Gateway
type TransferReq struct {
	PrimaryAccountNumber     string `json:"primaryAccountNumber"`
	ProcessingCode           string `json:"processingCode"`
	TransactionId            string `json:"transactionId"`
	BranchId                 string `json:"branchId"`
	InitiatorId              string `json:"initiatorId"`
	TransmissionDateTime     string `json:"transmissionDateTime"`
	CardAcceptorTerminalIp   string `json:"cardAcceptorTerminalIp"`
	CardAcceptorTerminalId   string `json:"cardAcceptorTerminalId"`
	SourceAccountNumber      string `json:"sourceAccountNumber"`
	SourceAccountName        string `json:"sourceAccountName"`
	DestinationAccountNumber string `json:"destinationAccountNumber"`
	DestinationAccountName   string `json:"destinationAccountName"`
	TransactionFee           string `json:"transactionFee"`
	TransactionAmount        string `json:"transactionAmount"`
	Currency                 string `json:"currency"`
}

// Response balik ke EDC (MTI 0610)
type HostResponse struct {
	Mti                      string `json:"mti"`
	PrimaryAccountNumber     string `json:"primaryAccountNumber"`
	ProcessingCode           string `json:"processingCode"`
	TransmissionDateTime     string `json:"transmissionDateTime"`
	SystemTraceAuditNumber   string `json:"systemTraceAuditNumber"`
	ResponseCode             string `json:"responseCode"`
	ResponseMessage          string `json:"responseMessage"`
	RetrievalReferenceNumber string `json:"retrievalReferenceNumber"`
	ApprovalCode             string `json:"approvalCode"`
	CardAcceptorMerchantId   string `json:"cardAcceptorMerchantId"`
	CardAcceptorTerminalId   string `json:"cardAcceptorTerminalId"`
	TransactionId            string `json:"transactionId"`
}
