package types

type CommandMessage struct {
	Command    string `json:"command"`
	CommandID  string `json:"commandID"`
	RoutingKey string `json:"routingKey"`
}

type CommandResponse struct {
	Response   string `json:"response"`
	CommandID  string `json:"commandID"`
	RoutingKey string `json:"routingKey"`
}
