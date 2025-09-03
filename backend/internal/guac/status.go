package guac

// Status represents the status of a Guacamole operation
type Status int

const (
	// Success indicates the operation was successful
	Success Status = iota

	// Unsupported indicates the requested operation is unsupported
	Unsupported

	// ServerError indicates an error on the server side
	ServerError

	// ServerBusy indicates the server is busy
	ServerBusy

	// UpstreamError indicates an error with an upstream server
	UpstreamError

	// UpstreamNotFound indicates an upstream server was not found
	UpstreamNotFound

	// UpstreamTimeout indicates an upstream server timed out
	UpstreamTimeout

	// UpstreamUnavailable indicates an upstream server is unavailable
	UpstreamUnavailable

	// ResourceNotFound indicates a resource was not found
	ResourceNotFound

	// ResourceClosed indicates a resource is closed
	ResourceClosed

	// ResourceConflict indicates a resource conflict
	ResourceConflict

	// SessionClosed indicates the session is closed
	SessionClosed

	// SessionConflict indicates a session conflict
	SessionConflict

	// SessionTimeout indicates the session timed out
	SessionTimeout

	// ClientBadRequest indicates a bad request from the client
	ClientBadRequest

	// ClientUnauthorized indicates the client is unauthorized
	ClientUnauthorized

	// ClientForbidden indicates the client is forbidden
	ClientForbidden

	// ClientTimeout indicates the client timed out
	ClientTimeout

	// ClientOverrun indicates the client sent too much data
	ClientOverrun

	// ClientBadType indicates the client sent data of an unsupported type
	ClientBadType

	// ClientTooMany indicates the client is using too many resources
	ClientTooMany
)

type statusData struct {
	name          string
	httpCode      int
	websocketCode int
	guacCode      int
}

func newStatusData(name string, httpCode, websocketCode, guacCode int) statusData {
	return statusData{
		name:          name,
		httpCode:      httpCode,
		websocketCode: websocketCode,
		guacCode:      guacCode,
	}
}

// String returns the string representation of the status
func (s Status) String() string {
	switch s {
	case Success:
		return "Success"
	case Unsupported:
		return "Unsupported"
	case ServerError:
		return "ServerError"
	case ServerBusy:
		return "ServerBusy"
	case UpstreamError:
		return "UpstreamError"
	case UpstreamNotFound:
		return "UpstreamNotFound"
	case UpstreamTimeout:
		return "UpstreamTimeout"
	case UpstreamUnavailable:
		return "UpstreamUnavailable"
	case ResourceNotFound:
		return "ResourceNotFound"
	case ResourceClosed:
		return "ResourceClosed"
	case ResourceConflict:
		return "ResourceConflict"
	case SessionClosed:
		return "SessionClosed"
	case SessionConflict:
		return "SessionConflict"
	case SessionTimeout:
		return "SessionTimeout"
	case ClientBadRequest:
		return "ClientBadRequest"
	case ClientUnauthorized:
		return "ClientUnauthorized"
	case ClientForbidden:
		return "ClientForbidden"
	case ClientTimeout:
		return "ClientTimeout"
	case ClientOverrun:
		return "ClientOverrun"
	case ClientBadType:
		return "ClientBadType"
	case ClientTooMany:
		return "ClientTooMany"
	default:
		return "Unknown"
	}
}
