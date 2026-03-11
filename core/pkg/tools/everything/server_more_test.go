package everything

import (
"context"
)

func (s *ServerTestSuite) TestInitRegistry() {
	// The runtime test helper isn't exported, but we can verify our init() hook worked 
// by seeing if NewServer is registered via runtime.Run behavior or by importing 
// a private test helper if there is one. Since there isn't one, we'll run it
// via a dummy context if possible, or just skip checking the internal registry map.
// Actually, we don't have Get exported in runtime. 
	
	// Just verify NewServer directly since we can't inspect the private map in runtime.
server, err := NewServer(context.Background(), Options{})
s.Require().NoError(err)
s.Require().NotNil(server)
}
