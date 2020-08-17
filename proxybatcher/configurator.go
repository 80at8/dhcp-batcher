package main

import (
	"crypto/rand"
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
)

var navDocsString = `
 Use your KEYBOARD to navigate 

Use the arrow keys ↑ ↓ to select
one of the menu options from the
list above.

 [TAB] switch input field
 [SAVE] save your changes
 [BACK] return to menu
 [SPACE] select field
`

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_-"
	bytes, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), nil
}

func configurator(loadYaml bool) {
	// try and load the yaml file, can't be found? lets make a new one.
	logger.Info("Load YAML is:", loadYaml)
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// VARIABLE DECLARATIONS (need to provision these here so we can build handlers)
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	var config programConfig
	var grid *tview.Grid
	var menuPage, configPage, batchRouterTablePage *tview.Pages
	if !loadYaml {
		options = programConfig{}
	}

	app := tview.NewApplication()

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// NAV DOCS
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	navDocs := tview.NewTextView()
	navDocs.SetBorderPadding(0, 1, 1, 1)
	navDocs.SetTitleAlign(tview.AlignCenter)
	navDocs.SetTitle(" Navigation Help")
	navDocs.SetTitleColor(tcell.ColorGreen)
	navDocs.SetBorder(false)
	fmt.Fprintf(navDocs, "%s", navDocsString)

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// MENU LIST
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	menuList := tview.NewList()

	menuList.AddItem("Operation Mode", "", 'o', func() {
		configPage.SwitchToPage("Operation Mode")
		batchRouterTablePage.HidePage("Batch Router Table")
		batchRouterTablePage.HidePage("DHCP Server IPs")
		app.SetFocus(configPage)
	})
	menuList.AddItem("Sonar Options", "", 's', func() {
		configPage.SwitchToPage("Sonar Options")
		batchRouterTablePage.HidePage("Batch Router Table")
		batchRouterTablePage.HidePage("DHCP Server IPs")
		app.SetFocus(configPage)
	})
	menuList.AddItem("Batch Options", "", 'b', func() {
		configPage.SwitchToPage("Batch Options")
		batchRouterTablePage.HidePage("Batch Router Table")
		app.SetFocus(configPage)
	})
	menuList.AddItem("Add/Remove Batch Routers", "", 'a', func() {
		configPage.SwitchToPage("Batch Router Options")
		batchRouterTablePage.ShowPage("Batch Router Table")
		batchRouterTablePage.HidePage("DHCP Server IPs")
		app.SetFocus(configPage)
	})
	menuList.AddItem("Proxy Options", "", 'p', func() {
		configPage.SwitchToPage("Proxy Options")
		batchRouterTablePage.ShowPage("DHCP Server IPs")
		batchRouterTablePage.HidePage("Batch Router Table")
		app.SetFocus(configPage)
	})
	menuList.AddItem("Logging Options", "", 'l', func() {
		configPage.SwitchToPage("Logging Options")
		batchRouterTablePage.HidePage("DHCP Server IPs")
		batchRouterTablePage.HidePage("Batch Router Table")
		app.SetFocus(configPage)
	})



	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// OPERATION MODE
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	operationModeForm := tview.NewForm()
	operationMode := tview.NewDropDown()
	operationMode.SetLabel("Operation Mode")

	operationMode.SetOptions([]string{"batch", "proxy"}, nil)
	operationModeForm.AddFormItem(operationMode)

	if loadYaml {
		if options.OperationMode == strings.ToLower("batch") {
			operationMode.SetCurrentOption(0)
		} else {
			operationMode.SetCurrentOption(1)
		}
	} else {
			operationMode.SetCurrentOption(0)
	}



	operationModeForm.AddButton("SAVE", func() {
		_, config.OperationMode = operationMode.GetCurrentOption()
		app.SetFocus(menuPage)
	})

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// SONAR OPTIONS
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	var sonarOptionsConfig sonarConfig
	versionIndex := 0
	apiUserName := ""
	apiKey := ""
	instanceURI := ""
	bearerToken := ""

	if loadYaml {
		sonarOptionsConfig = options.Sonar
		if sonarOptionsConfig.Version == 1 {
			versionIndex = 0
		} else {
			versionIndex = 1
		}

		apiUserName = sonarOptionsConfig.ApiUsername
		apiKey = sonarOptionsConfig.ApiKey
		instanceURI = sonarOptionsConfig.InstanceName
		bearerToken = sonarOptionsConfig.BearerToken

	}

	sonarOptionsForm := tview.NewForm()


	sonarOptionsForm.AddDropDown("Version", []string{"Sonar v1", "Sonar v2"}, versionIndex, func(option string, optionIndex int) {
		if optionIndex == 0 {
			sonarOptionsConfig.Version = 1
		} else {
			sonarOptionsConfig.Version = 2
		}
	})

	sonarOptionsForm.AddInputField("API Username", apiUserName, 256, nil, func(text string) {
		sonarOptionsConfig.ApiUsername = text
	})
	x := sonarOptionsForm.GetFormItemByLabel("API Username")
	fmt.Printf(x.GetLabel())
	sonarOptionsForm.AddInputField("API Key", apiKey, 256, nil, func(text string) {
		sonarOptionsConfig.ApiKey = text
	})
	sonarOptionsForm.AddInputField("Instance URI", instanceURI, 256, nil, func(text string) {
		sonarOptionsConfig.InstanceName = text
	})
	sonarOptionsForm.AddInputField("v2 Bearer Token", bearerToken, 0, nil, func(text string) {
		sonarOptionsConfig.BearerToken = text
	})

	sonarOptionsForm.AddButton("SAVE", func() {
		config.Sonar = sonarOptionsConfig
		sonarOptionsForm.SetFocus(0)
		app.SetFocus(menuPage)
	})

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// BATCH OPTIONS
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	var batchOptionsConfig batchConfig
	enableTls := 1
	tlsKey := "./certs/proxybatcher.key"
	tlsCert := "./certs/proxybatcher.crt"
	batcherIPAddress := "127.0.0.1"
	batcherHTTPPort := "80"
	batcherTLSPort := "443"
	batcherCycleTime := "5"
	if loadYaml {

		batchOptionsConfig = options.Batch
		if batchOptionsConfig.IsTLSEnabled {
			enableTls = 0
		} else {
			enableTls = 1
		}
		tlsKey = batchOptionsConfig.TlsKey
		tlsCert = batchOptionsConfig.TlsCert

		if tlsKey == "" {
			tlsKey = "./certs/proxybatcher.key"

		}

		if tlsCert == "" {
			tlsCert = "./certs/proxybatcher.crt"

		}

		batcherIPAddress = batchOptionsConfig.ServerIP
		batcherHTTPPort = batchOptionsConfig.HttpServerPort

		if batcherHTTPPort == "" {
			batcherHTTPPort = "80"
		}


		batcherTLSPort = batchOptionsConfig.TlsServerPort

		if batcherTLSPort == "" {
			batcherTLSPort = "443"
		}

		batcherCycleTime = strconv.Itoa(batchOptionsConfig.SchedulerCycleTime)

	}

	batchOptionsForm := tview.NewForm()
	tlsDropDown := tview.NewDropDown()
	tlsDropDown.SetLabel("Enable TLS")
	tlsDropDown.SetOptions([]string{"no","yes"}, func(text string, index int) {
		batchOptionsConfig.IsTLSEnabled = !(index == 0)
	})
	tlsDropDown.SetCurrentOption(enableTls)
	batchOptionsForm.AddFormItem(tlsDropDown)


	tlsKeyPath := tview.NewInputField()
	tlsKeyPath.SetLabel("TLS Key Path")
	tlsKeyPath.SetText(tlsKey)
	tlsKeyPath.SetFieldWidth(256)
	tlsKeyPath.SetChangedFunc(func(text string) {
		batchOptionsConfig.TlsKey = text
	})
	batchOptionsForm.AddFormItem(tlsKeyPath)

	tlsCertPath := tview.NewInputField()
	tlsCertPath.SetLabel("TLS Cert Path")
	tlsCertPath.SetText(tlsCert)
	tlsCertPath.SetFieldWidth(256)
	tlsCertPath.SetChangedFunc(func(text string) {
		batchOptionsConfig.TlsCert = text
	})
	batchOptionsForm.AddFormItem(tlsCertPath)

	batchIPAddress := tview.NewInputField()
	batchIPAddress.SetLabel("Batcher IP Address")
	batchIPAddress.SetText(batcherIPAddress)
	batchIPAddress.SetFieldWidth(16)
	batchOptionsForm.AddFormItem(batchIPAddress)


	batchHTTPPort := tview.NewInputField()
	batchHTTPPort.SetLabel("Batch HTTP Port")
	batchHTTPPort.SetText(batcherHTTPPort)
	batchHTTPPort.SetFieldWidth(5)
	batchHTTPPort.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
		if _, err := strconv.Atoi(textToCheck); err != nil {
			return false
		}
		return true
	})
	batchHTTPPort.SetChangedFunc(func(text string) {
		batchOptionsConfig.HttpServerPort = text
	})
	batchOptionsForm.AddFormItem(batchHTTPPort)

	batchTLSPort := tview.NewInputField()
	batchTLSPort.SetLabel("Batch TLS Port")
	batchTLSPort.SetText(batcherTLSPort)
	batchTLSPort.SetFieldWidth(5)
	batchTLSPort.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
		if _, err := strconv.Atoi(textToCheck); err != nil {
			return false
		}
		return true
	})
	batchTLSPort.SetChangedFunc(func(text string) {
		batchOptionsConfig.TlsServerPort = text
	})
	batchOptionsForm.AddFormItem(batchTLSPort)


	batchTime := tview.NewInputField()
	batchTime.SetLabel("Batch Cycle Time (minutes)")
	batchTime.SetText(batcherCycleTime)
	batchTime.SetFieldWidth(5)
	batchTime.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
		if _, err := strconv.Atoi(textToCheck); err != nil {
			return false
		}
		return true
	})
	batchTime.SetChangedFunc(func(text string) {
		if cycleTime, err := strconv.Atoi(text); err == nil {
			batchOptionsConfig.SchedulerCycleTime = cycleTime
			return
		}
		batchOptionsConfig.SchedulerCycleTime = 5
		return
	})
	batchOptionsForm.AddFormItem(batchTime)

	batchOptionsForm.AddButton("SAVE", func() {
		//if config.OperationMode == "batch" {
			if err := net.ParseIP(batchIPAddress.GetText()); err == nil {
				batchIPAddress.SetLabelColor(tcell.ColorRed)
				batchIPAddress.SetLabel("[red]Batcher IP Address")
				batchIPAddress.SetText(batchIPAddress.GetText())
				app.SetFocus(batchIPAddress)
			} else {
				batchIPAddress.SetLabel("Batcher IP Address")
				//fmt.Printf("%+v",batchOptionsConfig)
				config.Batch = batchOptionsConfig
				app.SetFocus(menuPage)
			}
		//}
	})

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// BATCH ROUTERS
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	var tmpRouter batchRouterAuth
	var tmpCfgRouters []batchRouterAuth
	batchRouterTable := tview.NewTable()

	if loadYaml {
		tmpCfgRouters = options.Batch.Routers

		batchRouterTable.SetTitle("Batch Routers")
		// Header Cells
		batchRouterTable.SetCell(0, 0, tview.NewTableCell("Router IP").
			SetTextColor(tcell.ColorYellow).SetSelectable(false))
		batchRouterTable.SetCell(0, 1, tview.NewTableCell("Username").
			SetTextColor(tcell.ColorYellow).SetSelectable(false))
		batchRouterTable.SetCell(0, 2, tview.NewTableCell("Password").
			SetTextColor(tcell.ColorYellow).SetSelectable(false))

		for k, v := range options.Batch.Routers {

			batchRouterTable.InsertRow(k+1).SetSelectable(true, false)
			batchRouterTable.SetCell(k+1, 0, tview.NewTableCell(v.RouterIP))
			batchRouterTable.SetCell(k+1, 1, tview.NewTableCell(v.Username))
			batchRouterTable.SetCell(k+1, 2, tview.NewTableCell(v.Password))
		}
		batchRouterTable.ScrollToBeginning()
	}

	batchRoutersOptionsForm := tview.NewForm()
	routerIP := tview.NewInputField()
	routerIP.SetLabel("Router IP")
	routerIP.SetText("")
	routerIP.SetFieldWidth(16)
	routerIP.SetChangedFunc(func(text string) {
		tmpRouter.RouterIP = text
	})
	routerUsername := tview.NewInputField()
	routerUsername.SetLabel("Script Username")
	routerUsername.SetText("")
	routerUsername.SetFieldWidth(32)
	routerUsername.SetChangedFunc(func(text string) {
		tmpRouter.Username = text
	})

	routerPassword := tview.NewInputField()
	routerPassword.SetLabel("Script Password")
	routerPassword.SetText("")
	routerPassword.SetFieldWidth(64)
	routerPassword.SetChangedFunc(func(text string) {
		tmpRouter.Password = text
	})
	batchRoutersOptionsForm.AddFormItem(routerIP)
	batchRoutersOptionsForm.AddFormItem(routerUsername)
	batchRoutersOptionsForm.AddFormItem(routerPassword)
	batchRoutersOptionsForm.AddButton("GENERATE PASSWORD", func() {
		str, err := GenerateRandomString(64)
		if err != nil {
			return
		} else {
			routerPassword.SetText(str)
			tmpRouter.Password = str
		}
	})

	batchRoutersOptionsForm.AddButton("ADD ROUTER", func() {
		if tmpRouter.RouterIP != "" && tmpRouter.Password != "" && tmpRouter.Username != "" {
			if err := net.ParseIP(tmpRouter.RouterIP); err != nil {
				routerIP.SetLabel("Router IP")

				tmpCfgRouters = append(tmpCfgRouters, tmpRouter)
				tmpRouter = batchRouterAuth{}

				// reset the form
				batchRoutersOptionsForm.Clear(false)

				// re-add the form input fields
				routerIP.SetText("")
				routerUsername.SetText("")
				routerPassword.SetText("")
				batchRoutersOptionsForm.AddFormItem(routerIP)
				batchRoutersOptionsForm.AddFormItem(routerUsername)
				batchRoutersOptionsForm.AddFormItem(routerPassword)
				batchRouterTable.Clear()

				batchRouterTable.SetTitle("Batch Routers")
				// Header Cells
				batchRouterTable.SetCell(0, 0, tview.NewTableCell("Router IP").
					SetTextColor(tcell.ColorYellow).SetSelectable(false))
				batchRouterTable.SetCell(0, 1, tview.NewTableCell("Username").
					SetTextColor(tcell.ColorYellow).SetSelectable(false))
				batchRouterTable.SetCell(0, 2, tview.NewTableCell("Password").
					SetTextColor(tcell.ColorYellow).SetSelectable(false))

				for k, v := range tmpCfgRouters {
					batchRouterTable.InsertRow(k+1).SetSelectable(true, false)
					batchRouterTable.SetCell(k+1, 0, tview.NewTableCell(v.RouterIP))
					batchRouterTable.SetCell(k+1, 1, tview.NewTableCell(v.Username))
					batchRouterTable.SetCell(k+1, 2, tview.NewTableCell(v.Password))
				}
				batchRouterTable.ScrollToBeginning()
			} else {
				routerIP.SetLabel("[red]Router IP")
				app.SetFocus(routerIP)
			}
		}
	})

	batchRoutersOptionsForm.AddButton("REMOVE ROUTER", func() {
		if batchRouterTable.GetRowCount() > 1 {
			batchRouterTable.SetSelectedFunc(func(row int, column int) {
				ip := batchRouterTable.GetCell(row, 0)
				index := 0
				found := false
				for k, v := range tmpCfgRouters {
					if v.RouterIP == ip.Text {
						found = true
						index = k
					}
				}
				if found {
					copy(tmpCfgRouters[index:], tmpCfgRouters[index+1:])    // Shift a[i+1:] left one index.
					tmpCfgRouters[len(tmpCfgRouters)-1] = batchRouterAuth{} // Erase last element (write zero value).
					tmpCfgRouters = tmpCfgRouters[:len(tmpCfgRouters)-1]
				}
				batchRouterTable.RemoveRow(row)
				app.SetFocus(configPage)
			})
			app.SetFocus(batchRouterTablePage)
		}
	})
	batchRoutersOptionsForm.AddButton("SAVE", func() {
		config.Batch.Routers = tmpCfgRouters
		app.SetFocus(menuList)
	})

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// PROXY OPTIONS
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	var tmpProxy proxyConfig
	proxyOptionsDHCPServersTable := tview.NewTable()

	interfaces := getSystemInterfaces()
	ipAddresses := getSystemIPAddresses()
	upstreamIndex, downstreamIndex, ipIndex := 0, 0, 0

	if loadYaml {

		for k, v := range interfaces {
			if v == options.Proxy.UpstreamInterface {
				upstreamIndex = k
			}
			if v == options.Proxy.DownstreamInterface {
				downstreamIndex = k
			}
		}
		for k, v := range ipAddresses {
			if v == options.Proxy.ProxyServerIP {
				ipIndex = k
			}
		}

		proxyOptionsDHCPServersTable.Clear()
		proxyOptionsDHCPServersTable.SetCell(0, 0, tview.NewTableCell("Upstream DHCP Server").
			SetTextColor(tcell.ColorYellow).SetSelectable(false))
		rem := 0
		for k, v := range options.Proxy.UpstreamServerIPs {
			if err := net.ParseIP(v); err != nil {
				proxyOptionsDHCPServersTable.InsertRow(k+1-rem).SetSelectable(true, false)
				proxyOptionsDHCPServersTable.SetCell(k+1-rem, 0, tview.NewTableCell(v).SetSelectable(true))
			} else {
				rem++
			}
		}
		proxyOptionsDHCPServersTable.ScrollToBeginning()
	}

	proxyOptionsForm := tview.NewForm()
	proxyOptionsForm.AddDropDown("Upstream Interface", getSystemInterfaces(), upstreamIndex, func(option string, optionIndex int) {
		tmpProxy.UpstreamInterface = option
	})
	proxyOptionsForm.AddDropDown("Downstream Interface", getSystemInterfaces(), downstreamIndex, func(option string, optionIndex int) {
		tmpProxy.DownstreamInterface = option
	})
	proxyOptionsForm.AddDropDown("Proxy Server IP", getSystemIPAddresses(), ipIndex, func(option string, optionIndex int) {
		tmpProxy.ProxyServerIP = option
	})

	// ADD Upstream DHCP server IPs
	proxyOptionsFormDHCPServerIP := tview.NewInputField()
	proxyOptionsFormDHCPServerIP.SetLabel("DHCP Server IP")
	proxyOptionsFormDHCPServerIP.SetFieldWidth(16)
	proxyOptionsForm.AddFormItem(proxyOptionsFormDHCPServerIP)
	proxyOptionsForm.AddButton("ADD DHCP SERVER", func() {
		text := proxyOptionsFormDHCPServerIP.GetText()
		if ip := net.ParseIP(text); ip != nil {
			isFound := false
			for _, v := range tmpProxy.UpstreamServerIPs {
				if v == text {
					isFound = true
				}
			}
			if !isFound {
				tmpProxy.UpstreamServerIPs = append(tmpProxy.UpstreamServerIPs, text)
			}
			proxyOptionsDHCPServersTable.Clear()
			proxyOptionsDHCPServersTable.SetCell(0, 0, tview.NewTableCell("Upstream DHCP Server").
				SetTextColor(tcell.ColorYellow).SetSelectable(false))
			for k, v := range tmpProxy.UpstreamServerIPs {
				proxyOptionsDHCPServersTable.InsertRow(k+1).SetSelectable(true, false)
				proxyOptionsDHCPServersTable.SetCell(k+1, 0, tview.NewTableCell(v).SetSelectable(true))
			}
			proxyOptionsDHCPServersTable.ScrollToBeginning()
		}
	})

	proxyOptionsForm.AddButton("REMOVE DHCP SERVER", func() {
		if proxyOptionsDHCPServersTable.GetRowCount() > 1 {
			proxyOptionsDHCPServersTable.SetSelectedFunc(func(row int, column int) {
				ip := proxyOptionsDHCPServersTable.GetCell(row, 0)
				index := 0
				found := false
				for k, v := range tmpProxy.UpstreamServerIPs {
					if v == ip.Text {
						found = true
						index = k
					}
				}
				if found {
					copy(tmpProxy.UpstreamServerIPs[index:], tmpProxy.UpstreamServerIPs[index+1:]) // Shift a[i+1:] left one index.
					tmpProxy.UpstreamServerIPs[len(tmpProxy.UpstreamServerIPs)-1] = ""             // Erase last element (write zero value).
					tmpProxy.UpstreamServerIPs = tmpProxy.UpstreamServerIPs[:len(tmpProxy.UpstreamServerIPs)-1]
				}
				proxyOptionsDHCPServersTable.RemoveRow(row)
				app.SetFocus(configPage)
			})
			app.SetFocus(proxyOptionsDHCPServersTable)
		}
	})

	proxyOptionsForm.AddButton("SAVE", func() {
		config.Proxy = tmpProxy
		app.SetFocus(menuList)
	})

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// LOGGING OPTIONS
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	var tmpLoggingConfig loggingConfig
	loggingOptionsForm := tview.NewForm()
	loggingMode := 0
	loggingFormat := 1
	loggingOutput := "./logs/proxybatcher.log"

	if loadYaml {
		tmpLoggingConfig = options.Logging
		switch strings.ToLower(tmpLoggingConfig.Mode) {
		case "debug":
			loggingMode = 0
		case "info":
			loggingMode = 1
		case "warn":
			loggingMode = 2
		}

		switch strings.ToLower(tmpLoggingConfig.Format) {
		case "text":
			loggingFormat = 0
		case "json":
			loggingFormat = 1
		}

		if strings.ToLower(tmpLoggingConfig.Output) == "" {
			loggingOutput = "./logs/proxybatcher.log"
		} else {
			loggingOutput = tmpLoggingConfig.Output
		}
	}

	loggingOptionsForm.AddDropDown("Logging Mode", []string{"debug", "info", "warn"}, loggingMode, func(option string, optionIndex int) {
		tmpLoggingConfig.Mode = option
	})
	loggingOptionsForm.AddDropDown("Logging Format", []string{"text", "json"}, loggingFormat, func(option string, optionIndex int) {
		tmpLoggingConfig.Format = option
	})
	loggingOptionsForm.AddInputField("Logging Output Path", loggingOutput, 0, nil, func(text string) {
		// TODO add path check
		tmpLoggingConfig.Output = text
	})
	loggingOptionsForm.AddButton("SAVE", func() {
		config.Logging = tmpLoggingConfig
		app.SetFocus(menuList)
	})

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// PAGE HANDLERS
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	menuPage = tview.NewPages()
	menuPage.AddPage("Options", menuList, true, true)

	configPage = tview.NewPages()
	configPage.AddPage("Operation Mode", operationModeForm, true, true)
	configPage.AddPage("Sonar Options", sonarOptionsForm, true, false)
	configPage.AddPage("Batch Options", batchOptionsForm, true, false)
	configPage.AddPage("Batch Router Options", batchRoutersOptionsForm, true, false)
	configPage.AddPage("Proxy Options", proxyOptionsForm, true, false)
	configPage.AddPage("Logging Options", loggingOptionsForm, true, false)

	batchRouterTablePage = tview.NewPages()
	batchRouterTablePage.AddPage("Batch Router Table", batchRouterTable, true, false)
	batchRouterTablePage.AddPage("DHCP Server IPs", proxyOptionsDHCPServersTable, true, false)

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// GRID TO PAGE MAPPINGS
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	grid = tview.NewGrid().
		SetBorders(true).
		SetColumns(-2, -4, -3).
		SetRows(0, -1, -1, -1, -1).
		AddItem(menuPage, 0, 0, 3, 1, 0, 0, true).
		AddItem(navDocs, 3, 0, 2, 1, 0, 0, false).
		AddItem(configPage, 0, 1, 2, 2, 0, 0, false).
		AddItem(batchRouterTablePage, 2, 1, 3, 2, 0, 0, true)


	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//
	// SAVE OPTIONS
	//
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////


	menuList.AddItem("SAVE AND QUIT", "", 'w', func() {
		options = config
		yamlFile, err := yaml.Marshal(options)
		if err != nil {
			fmt.Printf("%+v", options)

			logger.Error(err.Error())
		} else {
			ioutil.WriteFile("./conf/proxybatcher.yaml", yamlFile, 755)
			if err != nil {
				logger.Error(err.Error())
			}

		}
		app.Stop()
	})
	menuList.AddItem("QUIT WITHOUT SAVING", "", 'q', func() {
		app.Stop()
	})
	menuList.SetHighlightFullLine(true)



	if err := app.SetRoot(grid, true).SetFocus(menuList).Run(); err != nil {
		panic(err)
	}
}

func getSystemInterfaces() []string {
	var err error
	var interfaces []net.Interface
	var ifString []string
	interfaces, err = net.Interfaces()

	if err == nil {
		for _, v := range interfaces {
			ifString = append(ifString, v.Name)
		}
		return ifString
	} else {
		ifString = append(ifString, "unable to get system interfaces")
		return ifString
	}
}

func getSystemIPAddresses() []string {
	var err error
	var interfaces []net.Addr
	var ifString []string
	interfaces, err = net.InterfaceAddrs()

	if err == nil {
		for _, v := range interfaces {
			ipAddr, _, err := net.ParseCIDR(v.String())
			if err == nil {
				ifString = append(ifString, ipAddr.String())
			}
		}
		return ifString
	} else {
		ifString = append(ifString, "unable to get system ips")
		return ifString
	}

}
