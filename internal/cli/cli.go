package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/go-faster/jx"

	"github.com/myyra/procountor/procountorapi"
)

const (
	defaultBaseURL        = "https://nemo.procountor.com/api"
	envBaseURL            = "PROCOUNTOR_BASE_URL"
	envAuthFile           = "PROCOUNTOR_AUTH_FILE"
	envUsername           = "PROCOUNTOR_USERNAME"
	envPassword           = "PROCOUNTOR_PASSWORD"
	defaultRequestTimeout = 30 * time.Second
	exitCodeInvalidUsage  = 2
)

type runner struct {
	in     io.Reader
	inFile *os.File
	lineIn *bufio.Reader
	out    io.Writer
	errOut io.Writer

	baseURL  string
	authFile string
	timeout  time.Duration
	verbose  bool
	output   outputMode

	client *procountorapi.Client
}

type staticSecurity struct {
	token string
}

func (s staticSecurity) BearerAuth(context.Context, procountorapi.OperationName) (procountorapi.BearerAuth, error) {
	if strings.TrimSpace(s.token) == "" {
		return procountorapi.BearerAuth{}, errors.New("missing access token")
	}
	return procountorapi.BearerAuth{Token: s.token}, nil
}

type usageError struct {
	message string
}

func (e *usageError) Error() string {
	return e.message
}

func invalidUsage(format string, args ...any) error {
	return &usageError{message: fmt.Sprintf(format, args...)}
}

func wrapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

type GlobalFlags struct {
	BaseURL  AbsoluteURL   `default:"https://nemo.procountor.com/api"               env:"PROCOUNTOR_BASE_URL"                  help:"Base URL for API requests." name:"base-url"`
	AuthFile string        `env:"PROCOUNTOR_AUTH_FILE"                              help:"Path to the stored auth-state JSON." name:"auth-file"`
	Timeout  time.Duration `default:"30s"                                           help:"HTTP timeout for API requests."      name:"timeout"`
	Verbose  bool          `help:"Print raw HTTP requests and responses to stderr." name:"verbose"                             short:"v"`
	Output   outputMode    `default:"human"                                         enum:"human,json,plain"                    help:"Success output format."     name:"output"`
}

func (f *GlobalFlags) Validate() error {
	if f.Timeout <= 0 {
		return invalidUsage("--timeout must be greater than 0")
	}
	return nil
}

type CLI struct {
	GlobalFlags `embed:""`

	Auth             AuthCmd             `cmd:"" help:"Log in, refresh, inspect, or clear local session state."`
	Attachments      AttachmentsCmd      `cmd:"" help:"Upload, download, rename, and delete invoice attachments."`
	BusinessPartners BusinessPartnersCmd `cmd:"" help:"Search and maintain customers, suppliers, and partner groups." name:"business-partners"`
	ChartOfAccounts  ChartOfAccountsCmd  `cmd:"" help:"Read the chart of accounts."                                   name:"chart-of-accounts"`
	Company          CompanyCmd          `cmd:"" help:"Read active company details."`
	CostCenters      CostCentersCmd      `cmd:"" help:"Read cost centers and linked ledger receipts."                 name:"cost-centers"`
	CostReceipts     CostReceiptsCmd     `cmd:"" help:"Search and mutate cost receipts."                              name:"cost-receipts"`
	Dimensions       DimensionsCmd       `cmd:"" help:"Manage dimensions and dimension items."`
	FiscalYears      FiscalYearsCmd      `cmd:"" help:"Read fiscal years."                                            name:"fiscal-years"`
	Invoices         InvoicesCmd         `cmd:"" help:"Search and mutate invoices, payment events, and comments."`
	LedgerReceipts   LedgerReceiptsCmd   `cmd:"" help:"Search and mutate ledger receipts and their transactions."     name:"ledger-receipts"`
	Persons          PersonsCmd          `cmd:"" help:"List, fetch, and update persons."`
	Products         ProductsCmd         `cmd:"" help:"List products."`
	Reports          ReportsCmd          `cmd:"" help:"Run accounting and ledger reports."`
	Vats             VATsCmd             `cmd:"" help:"Read VAT defaults and country/company settings."               name:"vats"`
	Schema           SchemaCmd           `cmd:"" help:"Print the JSON request-body schema for a command."`
}

type ExecuteHooks struct{}

type exitPanic struct {
	code int
}

func Execute(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	r := newRunner(in, out, errOut, ExecuteHooks{})
	return executeRunner(ctx, r, args)
}

func ExecuteWithHooks(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer, hooks ExecuteHooks) int {
	r := newRunner(in, out, errOut, hooks)
	return executeRunner(ctx, r, args)
}

func normalizeRootArgs(args []string) []string {
	if len(args) == 0 {
		return []string{"--help"}
	}
	return args
}

func newRunner(in io.Reader, out, errOut io.Writer, _ ExecuteHooks) *runner {
	r := &runner{
		in:     in,
		lineIn: bufio.NewReader(in),
		out:    out,
		errOut: errOut,
	}
	if f, ok := in.(*os.File); ok {
		r.inFile = f
	}
	return r
}

func executeRunner(ctx context.Context, r *runner, args []string) (code int) {
	parser, cli, err := newParser(r, false)
	if err != nil {
		r.writeError(err)
		return exitCode(err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			if ep, ok := recovered.(exitPanic); ok {
				code = ep.code
				return
			}
			panic(recovered)
		}
	}()

	kctx, parseErr := parser.Parse(normalizeRootArgs(args))
	if parseErr != nil {
		r.writeError(parseErr)
		return exitCode(parseErr)
	}

	r.applyGlobalFlags(cli.GlobalFlags)
	kctx.BindTo(ctx, (*context.Context)(nil))
	kctx.Bind(r)
	kctx.Bind(&cli.GlobalFlags)

	runErr := kctx.Run()
	if runErr != nil {
		r.writeError(runErr)
		return exitCode(runErr)
	}
	return 0
}

func exitCode(err error) int {
	var uErr *usageError
	if errors.As(err, &uErr) {
		return exitCodeInvalidUsage
	}
	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		return exitCodeInvalidUsage
	}
	return 1
}

func RenderHelpForDocs(args []string) (helpText string, err error) {
	var out strings.Builder
	r := &runner{
		in:     strings.NewReader(""),
		lineIn: bufio.NewReader(strings.NewReader("")),
		out:    &out,
		errOut: &out,
	}

	parser, _, err := newParser(r, true)
	if err != nil {
		return "", fmt.Errorf("build docs parser: %w", err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			if ep, ok := recovered.(exitPanic); ok && ep.code == 0 {
				helpText = out.String()
				return
			}
			panic(recovered)
		}
	}()

	if _, err := parser.Parse(append(args, "--help")); err != nil {
		return "", fmt.Errorf("render docs help: %w", err)
	}
	return out.String(), nil
}

func formatOptionalFlagHelp(value *kong.Value) string {
	formattedValue := *value
	if enumHelp := formatEnumHelp(value.EnumSlice()); enumHelp != "" {
		formattedValue.Help = appendHelpSuffix(value.Help, enumHelp)
	}

	help := kong.DefaultHelpValueFormatter(&formattedValue)
	if value.Flag == nil || value.Required {
		return help
	}
	return appendHelpSuffix(help, "(optional)")
}

func formatEnumHelp(values []string) string {
	formattedValues := formatHelpList(values)
	if formattedValues == "" {
		return ""
	}
	return "Possible values: " + formattedValues
}

func formatHelpList(values []string) string {
	const pairCount = 2

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		filtered = append(filtered, value)
	}
	switch len(filtered) {
	case 0:
		return ""
	case 1:
		return filtered[0]
	case pairCount:
		return filtered[0] + " or " + filtered[1]
	default:
		return strings.Join(filtered[:len(filtered)-1], ", ") + ", or " + filtered[len(filtered)-1]
	}
}

func appendHelpSuffix(help, suffix string) string {
	switch {
	case help == "":
		return suffix
	case strings.HasSuffix(help, "."):
		return help[:len(help)-1] + " " + suffix + "."
	default:
		return help + " " + suffix
	}
}

func (r *runner) writeError(err error) {
	var apiErr *procountorapi.ErrorResponseStatusCode
	if errors.As(err, &apiErr) {
		writeHumanError(r.errOut, formatAPIError(apiErr.StatusCode, apiErr.Response))
		return
	}

	var uErr *usageError
	if errors.As(err, &uErr) {
		writeHumanError(r.errOut, "invalid usage: "+uErr.message)
		return
	}

	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		writeHumanError(r.errOut, "invalid usage: "+parseErr.Error())
		return
	}

	writeHumanError(r.errOut, err.Error())
}

func newParser(r *runner, docsMode bool) (*kong.Kong, *CLI, error) {
	cli := &CLI{}
	description := strings.Join([]string{
		"Run Procountor accounting workflows from the terminal.",
		"Authentication for API commands is resolved from the stored auth state file and automatically refreshed when needed.",
		"Success output defaults to human-readable text; use --output human|json|plain when you need machine-friendly output.",
	}, "\n\n")

	parser, err := kong.New(
		cli,
		kong.Name("procountor"),
		kong.Description(description),
		kong.ValueFormatter(formatOptionalFlagHelp),
		kong.Writers(r.out, r.errOut),
		kong.Exit(func(code int) { panic(exitPanic{code: code}) }),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("build kong parser: %w", err)
	}
	if docsMode {
		parser.Stdout = r.out
		parser.Stderr = r.errOut
	}
	return parser, cli, nil
}

func (r *runner) applyGlobalFlags(flags GlobalFlags) {
	r.baseURL = string(flags.BaseURL)
	r.authFile = flags.AuthFile
	r.timeout = flags.Timeout
	r.verbose = flags.Verbose
	r.output = flags.Output
}

func (r *runner) requireAPI(ctx context.Context) error {
	return r.initClient(ctx)
}

func (r *runner) initClient(ctx context.Context) error {
	if r.client != nil {
		return nil
	}
	if strings.TrimSpace(r.baseURL) == "" {
		return invalidUsage("missing base URL: set --base-url or %s", envBaseURL)
	}

	token, err := r.resolveAccessToken(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return invalidUsage("not logged in: run 'procountor auth login'")
	}

	httpClient, err := r.newHTTPClient(false)
	if err != nil {
		return err
	}
	client, err := procountorapi.NewClient(
		r.baseURL,
		staticSecurity{token: token},
		procountorapi.WithClient(httpClient),
	)
	if err != nil {
		return fmt.Errorf("create API client: %w", err)
	}
	r.client = client
	return nil
}

func (r *runner) newHTTPClient(withCookieJar bool) (*http.Client, error) {
	timeout := r.timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	httpClient := &http.Client{Timeout: timeout}
	if r.verbose {
		httpClient.Transport = newVerboseRoundTripper(http.DefaultTransport, r.errOut)
	}
	if withCookieJar {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("create cookie jar: %w", err)
		}
		httpClient.Jar = jar
	}
	return httpClient, nil
}

type ogenEncoder interface {
	Encode(enc *jx.Encoder)
}

func marshalOutputValue(v any) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	if enc, ok := v.(ogenEncoder); ok {
		var e jx.Encoder
		enc.Encode(&e)
		return e.Bytes(), nil
	}

	rv := reflect.ValueOf(v)
	if rv.IsValid() && rv.Kind() != reflect.Pointer {
		pv := reflect.New(rv.Type())
		pv.Elem().Set(rv)
		if enc, ok := pv.Interface().(ogenEncoder); ok {
			var e jx.Encoder
			enc.Encode(&e)
			return e.Bytes(), nil
		}
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil, wrapErr("marshal JSON output", err)
	}
	return b, nil
}

func normalizeOutputValue(v any) (any, error) {
	b, err := marshalOutputValue(v)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var normalized any
	if err := dec.Decode(&normalized); err != nil {
		return nil, wrapErr("decode output for text rendering", err)
	}
	return normalized, nil
}

func decodeBody(data []byte, decodeFn func(*jx.Decoder) error) error {
	d := jx.DecodeBytes(data)
	if err := decodeFn(d); err != nil {
		return err
	}
	if err := d.Skip(); !errors.Is(err, io.EOF) {
		return errors.New("unexpected trailing data")
	}
	return nil
}

func writeIndent(sb *strings.Builder, indent int) {
	for range indent {
		sb.WriteByte(' ')
	}
}
