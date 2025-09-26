package main
import (
    "bufio"
    "encoding/csv"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net"
    "os"
    "path/filepath"
    "runtime"
    "sort"
    "strconv"
    "strings"
    "sync"
    "time"
    "github.com/xuri/excelize/v2"
)
// Result 存储扫描扫描结果结构
type Result struct {
    Timestamp time.Time `json:"time"`
    IP        string    `json:"ip"`
    Port      string    `json:"port"`
    Status    string    `json:"status"`
}
var (
    timeout     time.Duration
    ports       string
    ips         string
    file        string
    netCIDR     string
    verbose     bool
    concurrent  int
    outputFile  string
    onlyAlive   bool
    onlyUnalive bool
    retries     int
    listenPorts string
    listenTime  int
)
// init 初始化命令行参数
func init() {
    flag.StringVar(&ips, "ip", "", "IP 地址,多个用逗号分隔")
    flag.StringVar(&netCIDR, "net", "", "指定网段,如:10.1.93.0/24")
    flag.StringVar(&ports, "p", "22", "端口或范围,如:22,1-100(默认22)")
    flag.DurationVar(&timeout, "t", 2*time.Second, "连接超时时间,默认2秒")
    flag.IntVar(&retries, "n", 1, "探测重试次数")
    flag.BoolVar(&onlyUnalive, "u", false, "只输出close状态")
    flag.BoolVar(&onlyAlive, "a", false, "只输出open状态")
    flag.IntVar(&concurrent, "c", 0, "并发数")
    flag.StringVar(&file, "f", "", "从文件读取IP(第一列)")
    flag.StringVar(&outputFile, "o", "", "输出文件路径(仅支持.json/.xlsx/.csv/.txt)")
    flag.StringVar(&listenPorts, "l", "", "在0.0.0.0启动监听端口,如8080,9000")
    flag.IntVar(&listenTime, "time", 30, "监听持续时间(秒)")
    flag.BoolVar(&verbose, "v", false, "启用详细日志")
    flag.Usage = func() {
        progName := filepath.Base(os.Args[0])
        fmt.Println("===========================================================")
        fmt.Println("程序名称:TCP端口扫描小工具  版本:v1.5-20250925@奔跑的老六")
        fmt.Println("使用示例:")
        fmt.Printf("  %s 127.0.0.1 22\n", progName)
        fmt.Printf("  %s -ip 127.0.0.1 -p 22\n", progName)
        fmt.Printf("  %s -net 10.1.93.0/24 -p 22,80 -a -v -o res.xlsx\n", progName)
        fmt.Printf("  %s -l 8080,9000 -time 60 -v\n", progName)
        fmt.Println("\n参数说明:")
        fmt.Printf("  位置参数1    (IP地址,优先级低于 -ip)\n")
        fmt.Printf("  位置参数2    (端口,优先级低于 -p,默认22)\n")
        fmt.Printf("  -ip string   (IP地址,多个用逗号分隔)\n")
        fmt.Printf("  -net string  (网段,如:10.1.93.0/24)\n")
        fmt.Printf("  -p string    (端口/范围,如:22,1-100,默认22)\n")
        fmt.Printf("  -t duration  (超时时间,默认2秒)\n")
        fmt.Printf("  -n int       (重试次数,默认1)\n")
        fmt.Printf("  -u           (只输出关闭状态)\n")
        fmt.Printf("  -a           (只输出开放状态)\n")
        fmt.Printf("  -c int       (并发数,默认CPU核心数)\n")
        fmt.Printf("  -f string    (从文件读取IP,第一列有效)\n")
        fmt.Printf("  -o string    (输出文件,支持.json/.xlsx/.csv/.txt)\n")
        fmt.Printf("  -l string    (监听端口,如8080,9000)\n")
        fmt.Printf("  -time int    (监听持续时间(秒),默认30)\n")
        fmt.Printf("  -v           (详细日志模式)\n")
        fmt.Println("===========================================================")
    }
}
// parseList 分割逗号分隔的字符串为列表
func parseList(s string) []string {
    var result []string
    for _, item := range strings.Split(s, ",") {
        trimmed := strings.TrimSpace(item)
        if trimmed != "" {
            result = append(result, trimmed)
        }
    }
    return result
}
// validateIP 验证IP地址有效性
func validateIP(ipStr string) string {
    trimmed := strings.Trim(ipStr, "[] \t")
    if net.ParseIP(trimmed) == nil {
        return ""
    }
    return trimmed
}
// parsePortRange 解析端口范围(如1-100)为端口列表
func parsePortRange(portStr string) ([]string, error) {
    var ports []string
    if strings.Contains(portStr, "-") {
        parts := strings.Split(portStr, "-")
        if len(parts) != 2 {
            return nil, fmt.Errorf("无效范围:%s(格式应为start-end)", portStr)
        }
        start, err1 := strconv.Atoi(parts[0])
        end, err2 := strconv.Atoi(parts[1])
        if err1 != nil || err2 != nil {
            return nil, fmt.Errorf("非数字:%s(端口必须为整数)", portStr)
        }
        if start < 1 || end > 65535 || start > end {
            return nil, fmt.Errorf("越界:%s(端口范围1-65535)", portStr)
        }
        for i := start; i <= end; i++ {
            ports = append(ports, strconv.Itoa(i))
        }
    } else {
        portNum, err := strconv.Atoi(portStr)
        if err != nil {
            return nil, fmt.Errorf("非数字:%s(端口必须为整数)", portStr)
        }
        if portNum < 1 || portNum > 65535 {
            return nil, fmt.Errorf("越界:%d(端口范围1-65535)", portNum)
        }
        ports = append(ports, portStr)
    }
    return ports, nil
}
// expandPorts 解析端口表达式为端口列表
func expandPorts(portSpec string) ([]string, error) {
    if portSpec == "" {
        return nil, fmt.Errorf("端口不能为空")
    }
    portStrs := parseList(portSpec)
    if len(portStrs) == 0 {
        return nil, fmt.Errorf("未找到有效端口")
    }
    seen := make(map[int]bool)
    var result []int
    for _, p := range portStrs {
        expanded, err := parsePortRange(p)
        if err != nil {
            if getVerbose() {
                log.Printf("警告:跳过无效端口配置 %s:%v", p, err)
            }
            continue
        }
        for _, ep := range expanded {
            num, _ := strconv.Atoi(ep)
            if !seen[num] {
                seen[num] = true
                result = append(result, num)
            }
        }
    }
    if len(result) == 0 {
        return nil, fmt.Errorf("未解析到有效端口")
    }
    sort.Ints(result)
    var strResult []string
    for _, r := range result {
        strResult = append(strResult, strconv.Itoa(r))
    }
    return strResult, nil
}
// readIPsFromFile 从文件读取IP地址列表
func readIPsFromFile(path string) ([]string, error) {
    if path == "" {
        return nil, fmt.Errorf("文件路径不能为空")
    }
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return nil, fmt.Errorf("文件不存在:%s", path)
    }
    f, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("无法打开文件 %s:%w", path, err)
    }
    defer func() {
        if err := f.Close(); err != nil {
            log.Printf("警告:关闭文件失败 %s:%v", path, err)
        }
    }()
    var ips []string
    scanner := bufio.NewScanner(f)
    lineNum := 0
    for scanner.Scan() {
        lineNum++
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        fields := strings.Fields(line)
        if len(fields) == 0 {
            continue
        }
        ip := validateIP(fields[0])
        if ip == "" {
            if getVerbose() {
                log.Printf("警告:跳过第 %d 行,无效IP:'%s'", lineNum, fields[0])
            }
            continue
        }
        ips = append(ips, ip)
    }
    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("读取文件时出错 %s:%w", path, err)
    }
    if len(ips) == 0 {
        return nil, fmt.Errorf("文件 %s 中未找到有效IP", path)
    }
    return ips, nil
}
// parseCIDR 解析CIDR网段为IP列表
func parseCIDR(cidr string) ([]string, error) {
    if cidr == "" {
        return nil, fmt.Errorf("网段不能为空")
    }
    _, ipnet, err := net.ParseCIDR(cidr)
    if err != nil {
        return nil, fmt.Errorf("无效网段格式 %s:%w", cidr, err)
    }
    var ips []string
    for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
        ips = append(ips, ip.String())
    }
    if len(ips) > 2 {
        ips = ips[1 :len(ips)-1]
    }
    if len(ips) == 0 {
        return nil, fmt.Errorf("网段 %s 中未找到有效IP", cidr)
    }
    return ips, nil
}
// incIP IP地址自增(用于遍历网段)
func incIP(ip net.IP) {
    for j := len(ip) - 1; j >= 0; j-- {
        ip[j]++
        if ip[j] > 0 {
            break
        }
    }
}
// isPortAvailable 检查端口是否可用
func isPortAvailable(port string) bool {
    ln, err := net.Listen("tcp", "0.0.0.0:"+port)
    if err != nil {
        return false
    }
    if err := ln.Close(); err != nil {
        log.Printf("警告:关闭监听端口 %s 失败:%v", port, err)
    }
    return true
}
// startServer 启动TCP监听服务
func startServer(port string, duration time.Duration, wg *sync.WaitGroup) {
    defer wg.Done()
    if !isPortAvailable(port) {
        log.Printf("错误:端口 %s 已被占用,无法监听", port)
        return
    }
    address := "0.0.0.0:" + port
    ln, err := net.Listen("tcp", address)
    if err != nil {
        log.Printf("错误:无法监听端口 %s:%v", address, err)
        return
    }
    defer func() {
        if err := ln.Close(); err != nil {
            log.Printf("警告:关闭监听端口 %s 失败:%v", address, err)
        }
    }()
    seconds := int(duration.Seconds())
    log.Printf("开始监听 %s,持续 %d 秒", address, seconds)
    if getVerbose() {
        go func() {
            for i := seconds - 1; i >= 0; i-- {
                time.Sleep(1 * time.Second)
                log.Printf("端口 %s 将在 %d 秒后关闭", port, i)
            }
        }()
    }
    timer := time.NewTimer(duration)
    defer timer.Stop()
    go func() {
        for {
            conn, err := ln.Accept()
            if err != nil {
                select {
                case <-timer.C:
                    return
                default:
                    if getVerbose() {
                        log.Printf("警告:接受连接失败 %s:%v", address, err)
                    }
                    return
                }
            }
            go func(c net.Conn) {
                defer func() {
                    if err := c.Close(); err != nil {
                        log.Printf("警告:关闭连接 %s 失败:%v", c.RemoteAddr(), err)
                    }
                }()
                if getVerbose() {
                    log.Printf("收到来自 %s 的连接", c.RemoteAddr().String())
                }
            }(conn)
        }
    }()
    <-timer.C
    log.Printf("监听结束 %s", address)
}
// worker 扫描工作协程
func worker(jobs <-chan [2]string, results chan<- Result, timeout time.Duration, retries int, wg *sync.WaitGroup) {
    defer wg.Done()
    for job := range jobs {
        ip, port := job[0], job[1]
        address := net.JoinHostPort(ip, port)
        var status = "close"
        var latestTime = time.Now().UTC()
        for i := 0; i < retries; i++ {
            conn, err := net.DialTimeout("tcp", address, timeout)
            latestTime = time.Now().UTC()
            if err == nil {
                if err := conn.Close(); err != nil {
                    if getVerbose() {
                        log.Printf("警告:关闭连接 %s 失败:%v", address, err)
                    }
                }
                status = "open"
                break
            }
            if getVerbose() && i < retries-1 {
                log.Printf("重试 %d/%d:%s 连接失败:%v", i+1, retries, address, err)
            }
        }
        results <- Result{
            Timestamp:latestTime,
            IP:ip,
            Port:port,
            Status:status,
        }
    }
}
// getOutputFormat 获取输出文件格式
func getOutputFormat(filename string) (string, error) {
    if filename == "" {
        return "stdout", nil
    }
    ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
    switch ext {
    case "json", "xlsx", "csv", "txt":
        return ext, nil
    default:
        return "", fmt.Errorf("不支持的文件格式:%s,仅支持 .json/.xlsx/.csv/.txt", ext)
    }
}
// getVerbose 判断是否启用详细日志
func getVerbose() bool {
    if verbose {
        return true
    }
    if listenPorts != "" {
        return listenTime <= 30
    }
    return false
}
// main 主函数
func main() {
    flag.Parse()
    var positionalArgs []string
    flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
    flagSet.SetOutput(os.Stderr)
    flagSet.StringVar(&ips, "ip", "", "")
    flagSet.StringVar(&netCIDR, "net", "", "")
    flagSet.StringVar(&ports, "p", "22", "")
    flagSet.DurationVar(&timeout, "t", 2*time.Second, "")
    flagSet.IntVar(&retries, "n", 1, "")
    flagSet.BoolVar(&onlyUnalive, "u", false, "")
    flagSet.BoolVar(&onlyAlive, "a", false, "")
    flagSet.IntVar(&concurrent, "c", 0, "")
    flagSet.StringVar(&file, "f", "", "")
    flagSet.StringVar(&outputFile, "o", "", "")
    flagSet.StringVar(&listenPorts, "l", "", "")
    flagSet.IntVar(&listenTime, "time", 30, "")
    flagSet.BoolVar(&verbose, "v", false, "")
    for i := 1; i < len(os.Args); i++ {
        arg := os.Args[i]
        if !strings.HasPrefix(arg, "-") {
            positionalArgs = append(positionalArgs, arg)
            continue
        }
        flagName := arg[1:]
        f := flagSet.Lookup(flagName)
        if f == nil {
            log.Fatalf("错误:未知参数 %s", arg)
        }
        if flagName != "v" && flagName != "a" && flagName != "u" {
            if i+1 >= len(os.Args) {
                log.Fatalf("错误:参数 %s 需要指定值", arg)
            }
            paramValue := os.Args[i+1]
            if paramValue == "" || strings.HasPrefix(paramValue, "-") {
                log.Fatalf("错误:参数 %s 的值无效或缺失", arg)
            }
            if err := flagSet.Parse([]string{arg, paramValue}); err != nil {
                log.Fatalf("错误:解析参数 %s 失败:%v", arg, err)
            }
            i++
        } else {
            if err := flagSet.Parse([]string{arg}); err != nil {
                log.Fatalf("错误:解析参数 %s 失败:%v", arg, err)
            }
        }
    }
    if len(positionalArgs) >= 1 && ips == "" {
        ips = positionalArgs[0]
    }
    if len(positionalArgs) >= 2 && ports == "22" {
        ports = positionalArgs[1]
    }
    if len(os.Args) == 1 || (len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help")) {
        flag.Usage()
        return
    }
    if listenPorts != "" {
        validPorts, err := expandPorts(listenPorts)
        if err != nil {
            log.Fatalf("错误:解析监听端口失败:%v", err)
        }
        if len(validPorts) == 0 {
            log.Fatal("错误:未找到有效监听端口")
        }
        if listenTime <= 0 {
            log.Fatal("错误:监听时间必须大于0")
        }
        if getVerbose() {
            log.Printf("监听模式:详细输出已开启(时间≤30秒或指定了-v)")
        }
        duration := time.Duration(listenTime) * time.Second
        var wg sync.WaitGroup
        log.Printf("启动本地监听,持续 %d 秒", listenTime)
        for _, port := range validPorts {
            wg.Add(1)
            go startServer(port, duration, &wg)
        }
        wg.Wait()
        log.Println("所有监听已结束")
        return
    }
    if onlyAlive && onlyUnalive {
        log.Fatal("错误:-a(只显示开放)和 -u(只显示关闭)不能同时使用")
    }
    outputFormat, err := getOutputFormat(outputFile)
    if err != nil {
        log.Fatalf("错误:输出文件格式无效:%v", err)
    }
    if getVerbose() {
        if outputFile == "" {
            log.Println("输出格式:控制台(Tab分隔)")
        } else {
            log.Printf("输出格式:%s(文件:%s)", outputFormat, outputFile)
        }
    }
    if concurrent <= 0 {
        concurrent = runtime.GOMAXPROCS(0)
    }
    maxConcurrent := runtime.NumCPU() * 10000
    if concurrent > maxConcurrent {
        log.Printf("警告:并发数过大(%d),自动限制为%d(CPU核心数×10000)", concurrent, maxConcurrent)
        concurrent = maxConcurrent
    }
    if getVerbose() {
        log.Printf("启动扫描:并发数=%d, 超时时间=%v, 重试次数=%d", concurrent, timeout, retries)
    }
    var allResults []Result
    var ipList []string
    if netCIDR != "" {
        for _, cidr := range parseList(netCIDR) {
            ipsFromCIDR, err := parseCIDR(cidr)
            if err != nil {
                if getVerbose() {
                    log.Printf("警告:跳过无效网段 %s:%v", cidr, err)
                }
                continue
            }
            if getVerbose() {
                log.Printf("加载网段 %s:共 %d 个IP", cidr, len(ipsFromCIDR))
            }
            ipList = append(ipList, ipsFromCIDR...)
        }
    }
    if file != "" {
        ipsFromFile, err := readIPsFromFile(file)
        if err != nil {
            log.Fatalf("错误:读取IP文件失败:%v", err)
        }
        ipList = append(ipList, ipsFromFile...)
    }
    if ips != "" {
        ipList = append(ipList, parseList(ips)...)
    }
    if len(ipList) == 0 {
        log.Fatal("错误:未提供任何有效IP或网段(使用-ip/-net/-f指定)")
    }
    uniqueIPs := make(map[string]bool)
    var validIPs []string
    for _, raw := range ipList {
        ip := validateIP(raw)
        if ip == "" {
            if getVerbose() {
                log.Printf("警告:跳过无效IP:%s", raw)
            }
            continue
        }
        if !uniqueIPs[ip] {
            uniqueIPs[ip] = true
            validIPs = append(validIPs, ip)
        }
    }
    if len(validIPs) == 0 {
        log.Fatal("错误:未找到有效IP地址")
    }
    validPorts, err := expandPorts(ports)
    if err != nil {
        log.Fatalf("错误:解析端口失败:%v", err)
    }
    if len(validPorts) == 0 {
        log.Fatal("错误:未找到有效端口")
    }
    if getVerbose() {
        log.Printf("扫描任务:共 %d 个IP, %d 个端口", len(validIPs), len(validPorts))
    }
    totalJobs := len(validIPs) * len(validPorts)
    if totalJobs == 0 {
        log.Fatal("错误:无扫描任务(IP或端口为空)")
    }
    jobs := make(chan [2]string, totalJobs)
    results := make(chan Result, totalJobs)
    var wg sync.WaitGroup
    for i := 0; i < concurrent; i++ {
        wg.Add(1)
        go worker(jobs, results, timeout, retries, &wg)
    }
    go func() {
        defer close(jobs)
        for _, ip := range validIPs {
            for _, port := range validPorts {
                jobs <- [2]string{ip, port}
            }
        }
    }()
    go func() {
        wg.Wait()
        close(results)
    }()
    for result := range results {
        if onlyAlive && result.Status != "open" {
            continue
        }
        if onlyUnalive && result.Status != "close" {
            continue
        }
        allResults = append(allResults, result)
    }
    switch outputFormat {
    case "json":
        f, err := os.Create(outputFile)
        if err != nil {
            log.Fatalf("错误:创建JSON文件失败 %s:%v", outputFile, err)
        }
        defer func() {
            if err := f.Close(); err != nil {
                log.Printf("警告:关闭JSON文件失败 %s:%v", outputFile, err)
            }
        }()
        encoder := json.NewEncoder(f)
        encoder.SetIndent("", "  ")
        if err := encoder.Encode(allResults); err != nil {
            log.Fatalf("错误:写入JSON文件失败 %s:%v", outputFile, err)
        }
        if getVerbose() {
            log.Printf("JSON结果已写入 %s", outputFile)
        }
    case "csv":
        f, err := os.Create(outputFile)
        if err != nil {
            log.Fatalf("错误:创建CSV文件失败 %s:%v", outputFile, err)
        }
        defer func() {
            f.Sync()
            if err := f.Close(); err != nil {
                log.Printf("警告:关闭CSV文件失败 %s:%v", outputFile, err)
            }
        }()
        writer := csv.NewWriter(f)
        defer writer.Flush()
        if err := writer.Write([]string{"时间", "IP", "端口", "状态"}); err != nil {
            log.Fatalf("错误:写入CSV表头失败:%v", err)
        }
        for _, res := range allResults {
            ts := res.Timestamp.Format("2006-01-02 15:04:05")
            if err := writer.Write([]string{ts, res.IP, res.Port, res.Status}); err != nil {
                log.Printf("警告:写入CSV记录失败:%v", err)
            }
        }
        if getVerbose() {
            log.Printf("CSV结果已写入 %s", outputFile)
        }
    case "xlsx":
        f := excelize.NewFile()
        defer func() {
            if err := f.Close(); err != nil {
                log.Printf("警告:关闭XLSX文件失败:%v", err)
            }
        }()
        sheetName := "扫描结果"
        index, err := f.NewSheet(sheetName)
        if err != nil {
            log.Fatalf("错误:创建XLSX工作表失败:%v", err)
        }
        headerStyle, err := f.NewStyle(&excelize.Style{
            Fill:excelize.Fill{
                Type:"pattern",
                Pattern:1,
                Color:[]string{"#FFFFCC"},
            },
            Alignment:&excelize.Alignment{
                Horizontal:"left",
                Vertical:"center",
            },
            Border:[]excelize.Border{
                {Type:"left", Style:1, Color:"#000000"},
                {Type:"top", Style:1, Color:"#000000"},
                {Type:"right", Style:1, Color:"#000000"},
                {Type:"bottom", Style:1, Color:"#000000"},
            },
        })
        if err != nil {
            log.Fatalf("错误:创建表头样式失败:%v", err)
        }
        contentStyle, err := f.NewStyle(&excelize.Style{
            Alignment:&excelize.Alignment{
                Horizontal:"left",
                Vertical:"center",
            },
            Border:[]excelize.Border{
                {Type:"left", Style:1, Color:"#000000"},
                {Type:"top", Style:1, Color:"#000000"},
                {Type:"right", Style:1, Color:"#000000"},
                {Type:"bottom", Style:1, Color:"#000000"},
            },
        })
        if err != nil {
            log.Fatalf("错误:创建内容样式失败:%v", err)
        }
        headers := []string{"时间", "IP", "端口", "状态"}
        for col, header := range headers {
            cell, _ := excelize.CoordinatesToCellName(col+1, 1)
            f.SetCellValue(sheetName, cell, header)
            f.SetCellStyle(sheetName, cell, cell, headerStyle)
        }
        for row, res := range allResults {
            ts := res.Timestamp.Format("2006-01-02 15:04:05")
            data := []interface{}{ts, res.IP, res.Port, res.Status}
            for col, val := range data {
                cell, _ := excelize.CoordinatesToCellName(col+1, row+2)
                f.SetCellValue(sheetName, cell, val)
                f.SetCellStyle(sheetName, cell, cell, contentStyle)
            }
        }
        if err := f.SetPanes(sheetName, &excelize.Panes{
            Freeze:true,
            Split:false,
            YSplit:1,
            ActivePane:"bottom",
        }); err != nil {
            log.Printf("警告:设置表头锁定失败:%v", err)
        }
        for col := 0; col < len(headers); col++ {
            colName, _ := excelize.ColumnNumberToName(col + 1)
            maxWidth := 0
            headerCell, _ := excelize.CoordinatesToCellName(col+1, 1)
            headerVal, _ := f.GetCellValue(sheetName, headerCell)
            if len(headerVal) > maxWidth {
                maxWidth = len(headerVal)
            }
            for row := 0; row < len(allResults); row++ {
                dataCell, _ := excelize.CoordinatesToCellName(col+1, row+2)
                dataVal, _ := f.GetCellValue(sheetName, dataCell)
                if len(dataVal) > maxWidth {
                    maxWidth = len(dataVal)
                }
            }
            if err := f.SetColWidth(sheetName, colName, colName, float64(maxWidth)*1.2); err != nil {
                log.Printf("警告:设置列宽失败:%v", err)
            }
        }
        f.SetActiveSheet(index)
        if err := f.SaveAs(outputFile); err != nil {
            log.Fatalf("错误:保存XLSX文件失败 %s:%v", outputFile, err)
        }
        if getVerbose() {
            log.Printf("XLSX结果已写入 %s", outputFile)
        }
    case "txt":
        f, err := os.Create(outputFile)
        if err != nil {
            log.Fatalf("错误:创建TXT文件失败 %s:%v", outputFile, err)
        }
        defer func() {
            f.Sync()
            if err := f.Close(); err != nil {
                log.Printf("警告:关闭TXT文件失败 %s:%v", outputFile, err)
            }
        }()
        writer := bufio.NewWriter(f)
        defer writer.Flush()
        fmt.Fprintf(writer, "时间\tIP\t端口\t状态\n")
        for _, res := range allResults {
            ts := res.Timestamp.Format("2006-01-02 15:04:05")
            if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", ts, res.IP, res.Port, res.Status); err != nil {
                log.Printf("警告:写入TXT记录失败:%v", err)
            }
        }
        if getVerbose() {
            log.Printf("TXT结果已写入 %s", outputFile)
        }
    case "stdout":
        writer := bufio.NewWriter(os.Stdout)
        defer writer.Flush()
        // fmt.Fprintf(writer, "时间\tIP\t端口\t状态\n")
        for _, res := range allResults {
            ts := res.Timestamp.Format("2006-01-02_15:04:05")
            if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", ts, res.IP, res.Port, res.Status); err != nil {
                log.Printf("警告:输出到控制台失败:%v", err)
            }
        }
    }
    if getVerbose() {
        log.Println("扫描完成")
    }
}