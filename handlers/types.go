package handlers

// CreateRequest - Create 엔드포인트 요청 구조체
type CreateRequest struct {
	ProvisionID string `json:"provision_id" binding:"required"`
	ServiceID   string `json:"service_id" binding:"required"`
	Category    string `json:"category" binding:"required"`
	Region      string `json:"region" binding:"required"`
	UID         string `json:"uid" binding:"required"`
}
