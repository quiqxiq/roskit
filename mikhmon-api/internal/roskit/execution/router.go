package execution

type RouterSession struct {
	RouterID string
	Conn     *RouterConn
}

func NewRouterSession(routerID string, conn *RouterConn) *RouterSession {
	return &RouterSession{RouterID: routerID, Conn: conn}
}
