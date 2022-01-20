package beater

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/atisu/wmibeat-ohm2/config"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// wmibeat-ohm2 configuration.
type wmibeatohm2 struct {
	beat   *beat.Beat
	done   chan struct{}
	config config.WmibeatConfig
	client beat.Client
	log    *logp.Logger
}

// New creates an instance of wmibeat-ohm2.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := config.DefaultConfig
	if err := b.BeatConfig.Unpack(&c); err != nil {
		return nil, fmt.Errorf("ERROR READING CONFIG FILE: %v", err)
	}

	log := logp.NewLogger("wmibeatohm2")

	bt := &wmibeatohm2{
		beat:   b,
		done:   make(chan struct{}),
		config: c,
		log:    log,
	}

	return bt, nil
}

// Run starts wmibeat-ohm2.
func (bt *wmibeatohm2) Run(b *beat.Beat) error {
	bt.log.Info("wmibeat-ohm2 is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(bt.config.Period)
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		err = func() error {
			ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)

			wmiscriptObj, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
			defer wmiscriptObj.Release()
			if err != nil {
				bt.log.Error("ERROR: oleutil.CreateObject(WbemScripting.SWbemLocator)")
				return err
			}
			wmiqi, err := wmiscriptObj.QueryInterface(ole.IID_IDispatch)
			defer wmiqi.Release()
			if err != nil {
				bt.log.Error("ERROR: QueryInterface(ole.IID_IDispatch)")
				return err
			}
			serviceObj, err := oleutil.CallMethod(wmiqi, "ConnectServer")
			defer serviceObj.Clear()
			if err != nil {
				bt.log.Error("ERROR: default ConnectServer()")
				return err
			}
			service := serviceObj.ToIDispatch()
			//defer service.Release()

			var allValues common.MapStr
			for _, class := range bt.config.Classes {
				if len(class.Fields) > 0 {
					var query bytes.Buffer
					wmiFields := class.Fields
					query.WriteString("SELECT ")
					query.WriteString(strings.Join(wmiFields, ","))
					query.WriteString(" FROM ")
					query.WriteString(class.Class)
					if class.WhereClause != "" {
						query.WriteString(" WHERE ")
						query.WriteString(class.WhereClause)
					}
					bt.log.Info("query: " + query.String())
					resultObj, err := oleutil.CallMethod(service, "ExecQuery", query.String())
					defer resultObj.Clear()
					if err != nil {
						bt.log.Error("cannot query class `" + class.Class + "`")
						continue
					}
					result := resultObj.ToIDispatch()
					defer result.Release()
					countObj, err := oleutil.GetProperty(result, "Count")
					defer countObj.Clear()
					if err != nil {
						bt.log.Error("cannot query count property for class `" + class.Class + "`")
						continue
					}
					count := int(countObj.Val)

					var classValues interface{} = nil

					if class.ObjectTitle != "" {
						classValues = common.MapStr{}
					} else {
						classValues = []common.MapStr{}
					}
					for i := 0; i < count; i++ {
						rowObj, err := oleutil.CallMethod(result, "ItemIndex", i)
						if err != nil {
							bt.log.Error("cannot call ItemIndex for class `" + class.Class + "`")
							continue
						}
						row := rowObj.ToIDispatch()
						defer rowObj.Clear()
						var rowValues common.MapStr
						var objectTitle = ""
						var hasError int = 0
						for _, j := range wmiFields {
							wmiObj, err := oleutil.GetProperty(row, j)
							defer wmiObj.Clear()

							if err != nil {
								bt.log.Error("cannot get property for class `" + class.Class + "`")
								hasError = 1
								break
							}
							var objValue = wmiObj.Value()
							if class.ObjectTitle == j {
								objectTitle = objValue.(string)
							}
							rowValues = common.MapStrUnion(rowValues, common.MapStr{j: objValue})

						}
						if hasError == 0 {
							if class.ObjectTitle != "" {
								if objectTitle != "" {
									classValues = common.MapStrUnion(classValues.(common.MapStr), common.MapStr{objectTitle: rowValues})
								} else {
									classValues = common.MapStrUnion(classValues.(common.MapStr), common.MapStr{strconv.Itoa(i): rowValues})
								}
							} else {
								classValues = append(classValues.([]common.MapStr), rowValues)
							}
						}
						rowValues = nil
					}
					allValues = common.MapStrUnion(allValues, common.MapStr{class.Class: classValues})
					classValues = nil

				} else {
					var errorString bytes.Buffer
					errorString.WriteString("No fields defined for class ")
					errorString.WriteString(class.Class)
					errorString.WriteString(".  Skipping")
					bt.log.Warn(errorString.String())
				}
			}

			for _, namespace := range bt.config.Namespaces {
				//bt.log.Info("Namespace: root\\" + namespace.Namespace)
				func() {
					nsServiceObj, err := oleutil.CallMethod(wmiqi, "ConnectServer", "localhost",
						"root\\"+namespace.Namespace)
					if err != nil {
						bt.log.Error("cannot connect to namespace `" + namespace.Namespace + "`, skipping it.")
						return
					}
					nsService := nsServiceObj.ToIDispatch()
					defer nsService.Release()

					var query bytes.Buffer
					metricNameFields := namespace.MetricNameCombinedFields
					var allWMIFields []string = append(metricNameFields, namespace.MetricValueField)
					query.WriteString("SELECT ")
					query.WriteString(strings.Join(allWMIFields, ","))
					query.WriteString(" FROM ")
					query.WriteString(namespace.Class)
					if namespace.WhereClause != "" {
						query.WriteString(" WHERE ")
						query.WriteString(namespace.WhereClause)
					}
					resultObj, err := oleutil.CallMethod(nsService, "ExecQuery", query.String())
					if err != nil {
						bt.log.Error("cannot exec query in current namespace, skipping it: " + query.String())
						return
					}
					result := resultObj.ToIDispatch()
					defer result.Release()

					countObj, err := oleutil.GetProperty(result, "Count")
					if err != nil {
						bt.log.Error("cannot get `Count` property, skipping current namespace.")
						return
					}
					defer countObj.Clear()
					count := int(countObj.Val)
					bt.log.Info("Namespace: root\\" + namespace.Namespace + ", count in class: " + strconv.Itoa(count))
					for i := 0; i < count; i++ {
						func() {
							rowObj, err := oleutil.CallMethod(result, "ItemIndex", i)
							defer rowObj.Clear()
							if err != nil {
								bt.log.Error("cannot fetch ItemIndex, skipping it.")
								return
							}
							row := rowObj.ToIDispatch()
							var objectTitle = namespace.Namespace + "_" + namespace.Class
							var metricValue = ""
							var hasError int = 0
							for _, j := range allWMIFields {
								func() {
									wmiObj, err := oleutil.GetProperty(row, j)
									defer wmiObj.Clear()
									if err != nil {
										bt.log.Error("cannot get `Count` property for row, skipping it.")
										hasError = 1
										return
									}
									var objValue = wmiObj.Value()
									if j != namespace.MetricValueField {
										objectTitle = fmt.Sprintf("%s_%v", objectTitle, objValue)
									} else {
										metricValue = fmt.Sprintf("%v", objValue)
									}
								}()
							}
							if hasError == 0 {
								objectTitle = strings.ReplaceAll(strings.ReplaceAll(objectTitle, " ", ""), "#", "")
								allValues = common.MapStrUnion(allValues, common.MapStr{objectTitle: metricValue})
							}
						}()
					}
				}()
			}

			ole.CoUninitialize()

			event := beat.Event{
				Timestamp: time.Now(),
				Fields: common.MapStr{
					"type": b.Info.Name,
					"wmi":  allValues,
				},
			}
			bt.client.Publish(event)
			allValues = nil
			return nil
		}()
		if err != nil {
			return err
		}
	}
}

// Stop stops wmibeat-ohm2.
func (bt *wmibeatohm2) Stop() {
	bt.client.Close()
	close(bt.done)
}
