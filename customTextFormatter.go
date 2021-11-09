package mflogger

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type CustomTextFormatter struct{}

const (
	red    = 31
	yellow = 33
	blue   = 36
	gray   = 37
)

func Init(devMode bool) {
	if !devMode {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&CustomTextFormatter{})
	}
}

func (f *CustomTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		if k != "error-stack" {
			keys = append(keys, k)
		}
	}

	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	var levelColor int
	switch entry.Level {
	case logrus.DebugLevel:
		levelColor = gray
	case logrus.WarnLevel:
		levelColor = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	default:
		levelColor = blue
	}

	levelText := strings.ToUpper(entry.Level.String())[0:4]

	fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%s] %s ", levelColor, levelText, entry.Time.Format("15:04:05"), entry.Message)

	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " \x1b[%dm%s\x1b[0m=", levelColor, k)
		f.appendValue(b, v)
	}

	b.WriteByte('\n')

	if stack, ok := entry.Data["error-stack"]; ok {
		if stack, ok := stack.(string); ok {
			fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m=", levelColor, "error-stack")
			b.WriteByte('\n')
			for _, r := range strings.Split(stack, `\n`) {
				b.WriteString(r)
				b.WriteByte('\n')
			}
		}
	}

	return b.Bytes(), nil
}

func (f *CustomTextFormatter) needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return true
		}
	}
	return false
}

func (f *CustomTextFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)
	b.WriteByte(' ')
}

func (f *CustomTextFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	if !f.needsQuoting(stringVal) {
		b.WriteString(stringVal)
	} else {
		b.WriteString(fmt.Sprintf("%q", stringVal))
	}
}
