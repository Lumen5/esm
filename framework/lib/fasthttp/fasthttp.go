package fasthttp

// Re-export everything from the actual fasthttp library
import "github.com/valyala/fasthttp"

type Client = fasthttp.Client
type Request = fasthttp.Request
type Response = fasthttp.Response

var AcquireRequest = fasthttp.AcquireRequest
var AcquireResponse = fasthttp.AcquireResponse
var ReleaseRequest = fasthttp.ReleaseRequest
var ReleaseResponse = fasthttp.ReleaseResponse
var WriteGzipLevel = fasthttp.WriteGzipLevel

const CompressBestSpeed = fasthttp.CompressBestSpeed
