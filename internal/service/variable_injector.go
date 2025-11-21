package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"test-management-service/internal/testcase"
)

// VariableInjector 变量注入器
type VariableInjector struct {
	envService EnvironmentService
}

// NewVariableInjector 创建变量注入器
func NewVariableInjector(envService EnvironmentService) *VariableInjector {
	return &VariableInjector{
		envService: envService,
	}
}

// GetActiveEnvironmentVariables 获取激活环境的变量
func (vi *VariableInjector) GetActiveEnvironmentVariables() (map[string]string, error) {
	activeEnv, err := vi.envService.GetActiveEnvironment()
	if err != nil {
		return nil, err
	}

	if activeEnv.Variables == nil {
		return make(map[string]string), nil
	}

	// Convert to map[string]string
	result := make(map[string]string)
	for k, v := range activeEnv.Variables {
		result[k] = vi.valueToString(v)
	}

	return result, nil
}

// InjectVariables 注入环境变量到配置中
// 支持三层变量优先级: envVars < workflowVars < inlineVars
func (vi *VariableInjector) InjectVariables(
	config interface{},
	workflowVars map[string]interface{},
) (interface{}, error) {
	// 1. 获取当前激活的环境变量
	activeEnv, err := vi.envService.GetActiveEnvironment()
	if err != nil {
		// 如果没有激活环境，使用空变量集
		return vi.injectWithVars(config, make(map[string]interface{}), workflowVars), nil
	}

	envVars := activeEnv.Variables
	if envVars == nil {
		envVars = make(map[string]interface{})
	}

	// 2. 执行变量注入
	return vi.injectWithVars(config, envVars, workflowVars), nil
}

// injectWithVars 使用指定的变量集进行注入
func (vi *VariableInjector) injectWithVars(
	config interface{},
	envVars map[string]interface{},
	workflowVars map[string]interface{},
) interface{} {
	// 合并变量（优先级: envVars < workflowVars）
	mergedVars := vi.mergeVariables(envVars, workflowVars)

	// 递归替换配置中的占位符
	return vi.replaceVariables(config, mergedVars)
}

// mergeVariables 合并变量，后者优先级更高
func (vi *VariableInjector) mergeVariables(
	base map[string]interface{},
	override map[string]interface{},
) map[string]interface{} {
	result := make(map[string]interface{})

	// 复制基础变量
	for k, v := range base {
		result[k] = v
	}

	// 覆盖变量
	if override != nil {
		for k, v := range override {
			result[k] = v
		}
	}

	return result
}

// replaceVariables 递归替换所有变量占位符
func (vi *VariableInjector) replaceVariables(
	value interface{},
	vars map[string]interface{},
) interface{} {
	switch v := value.(type) {
	case string:
		return vi.replaceStringVariables(v, vars)

	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = vi.replaceVariables(val, vars)
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = vi.replaceVariables(val, vars)
		}
		return result

	default:
		return value
	}
}

// replaceStringVariables 替换字符串中的变量占位符
// 支持格式: {{VAR_NAME}}
func (vi *VariableInjector) replaceStringVariables(
	str string,
	vars map[string]interface{},
) interface{} {
	// 正则匹配 {{VAR_NAME}}
	re := regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

	matches := re.FindAllStringSubmatch(str, -1)
	if len(matches) == 0 {
		return str
	}

	// 如果整个字符串就是一个变量引用，直接返回变量值（保持类型）
	if len(matches) == 1 && matches[0][0] == str {
		varName := matches[0][1]
		if val, exists := vars[varName]; exists {
			return val
		}
		return str // 变量不存在，返回原字符串
	}

	// 字符串包含多个变量或混合内容，替换为字符串
	result := str
	for _, match := range matches {
		placeholder := match[0] // {{VAR_NAME}}
		varName := match[1]     // VAR_NAME

		if val, exists := vars[varName]; exists {
			// 将变量值转换为字符串
			replacement := vi.valueToString(val)
			result = strings.ReplaceAll(result, placeholder, replacement)
		}
	}

	return result
}

// valueToString 将任意值转换为字符串
func (vi *VariableInjector) valueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return strconv.FormatFloat(v.(float64), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		// 复杂类型转为 JSON 字符串
		bytes, _ := json.Marshal(v)
		return string(bytes)
	}
}

// InjectHTTPVariables 注入变量到 HTTP 配置
func (vi *VariableInjector) InjectHTTPVariables(httpConfig *testcase.HTTPTest) error {
	if httpConfig == nil {
		return nil
	}

	// Convert HTTP config to map
	configMap := map[string]interface{}{
		"method":  httpConfig.Method,
		"path":    httpConfig.Path,
		"headers": httpConfig.Headers,
		"body":    httpConfig.Body,
	}

	// Inject variables
	injected, err := vi.InjectVariables(configMap, nil)
	if err != nil {
		return err
	}

	// Convert back to HTTPTest
	injectedMap := injected.(map[string]interface{})
	if method, ok := injectedMap["method"].(string); ok {
		httpConfig.Method = method
	}
	if path, ok := injectedMap["path"].(string); ok {
		httpConfig.Path = path
	}
	if headers, ok := injectedMap["headers"].(map[string]interface{}); ok {
		// Convert map[string]interface{} to map[string]string
		stringHeaders := make(map[string]string)
		for k, v := range headers {
			if str, ok := v.(string); ok {
				stringHeaders[k] = str
			} else {
				stringHeaders[k] = vi.valueToString(v)
			}
		}
		httpConfig.Headers = stringHeaders
	}
	if body, ok := injectedMap["body"].(map[string]interface{}); ok {
		httpConfig.Body = body
	}

	return nil
}

// InjectCommandVariables 注入变量到命令配置
func (vi *VariableInjector) InjectCommandVariables(commandConfig *testcase.CommandTest) error {
	if commandConfig == nil {
		return nil
	}

	// Convert Command config to map
	configMap := map[string]interface{}{
		"cmd":  commandConfig.Cmd,
		"args": commandConfig.Args,
	}

	// Inject variables
	injected, err := vi.InjectVariables(configMap, nil)
	if err != nil {
		return err
	}

	// Convert back to CommandTest
	injectedMap := injected.(map[string]interface{})
	if cmd, ok := injectedMap["cmd"].(string); ok {
		commandConfig.Cmd = cmd
	}
	if args, ok := injectedMap["args"].([]interface{}); ok {
		// Convert []interface{} to []string
		stringArgs := make([]string, len(args))
		for i, arg := range args {
			if str, ok := arg.(string); ok {
				stringArgs[i] = str
			} else {
				stringArgs[i] = fmt.Sprintf("%v", arg)
			}
		}
		commandConfig.Args = stringArgs
	}

	return nil
}

// InjectIntoHTTPConfig 注入变量到 HTTP 配置 (map版本)
func (vi *VariableInjector) InjectIntoHTTPConfig(
	httpConfig map[string]interface{},
	workflowVars map[string]interface{},
) (map[string]interface{}, error) {
	injected, err := vi.InjectVariables(httpConfig, workflowVars)
	if err != nil {
		return nil, err
	}

	return injected.(map[string]interface{}), nil
}

// InjectIntoCommandConfig 注入变量到命令配置 (map版本)
func (vi *VariableInjector) InjectIntoCommandConfig(
	commandConfig map[string]interface{},
	workflowVars map[string]interface{},
) (map[string]interface{}, error) {
	injected, err := vi.InjectVariables(commandConfig, workflowVars)
	if err != nil {
		return nil, err
	}

	return injected.(map[string]interface{}), nil
}
