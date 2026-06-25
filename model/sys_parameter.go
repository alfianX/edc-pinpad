package model

// SysParameter merepresentasikan tabel 'sys_parameters' di MySQL
type SysParameter struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	ParamKey    string `gorm:"type:varchar(50);uniqueIndex:idx_module_param" json:"param_key"`
	ParamType   string `gorm:"type:varchar(20);not null" json:"param_type"`
	ParamValue  string `gorm:"type:text;not null" json:"param_value"`
	Description string `gorm:"type:text" json:"description"`
}
