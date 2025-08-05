package model

type StartBotRequest struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	OfflineUsername string `json:"offlineUsername"`
	// TODO add online auth
}
