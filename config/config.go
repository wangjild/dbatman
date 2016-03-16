package config

import (
	"fmt"
	"github.com/bytedance/dbatman/Godeps/_workspace/src/gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

var conf CONF

type CONF struct {
	path             string
	lastModifiedTime time.Time
	mu               sync.RWMutex
	proxyConfig      *ProxyConfig
}

type ProxyConfig struct {
	Global   *GlobalConfig             `yaml:"global"`
	Clusters map[string]*ClusterConfig `yaml:"clusters"`
	Users    map[string]*UserConfig    `yaml:"users"`
}

type GlobalConfig struct {
	Port              int
	ManagePort        int      `yaml:"manage_port"`
	MaxConnections    int      `yaml:"max_connections"`
	LogFilename       string   `yaml:"log_filename"`
	LogLevel          int      `yaml:"log_level"`
	LogMaxSize        int      `yaml:"log_maxsize"`
	ClientTimeout     int      `yaml:"client_timeout"`
	ServerTimeout     int      `yaml:"server_timeout"`
	WriteTimeInterval int      `yaml:"write_time_interval"`
	ConfAutoload      int      `yaml:"conf_autoload"`
	AuthIPs           []string `yaml:"auth_ips"`
}

type ClusterConfig struct {
	Master *NodeConfig
	Slaves []*NodeConfig
}

type NodeConfig struct {
	Host                  string
	Port                  int
	Username              string
	Password              string
	DBName                string
	Weight                int
	MaxConnections        int `yaml:"max_connections"`
	MaxConnectionPoolSize int `yaml:"max_connection_pool_size"`
	ConnectTimeout        int `yaml:"connect_timeout"`
	TimeReconnectInterval int `yaml:"time_reconnect_interval"`
}

type UserConfig struct {
	Username       string
	Password       string
	DBName         string
	Charset        string
	MaxConnections int      `yaml:"max_connections"`
	MinConnections int      `yaml:"min_connections"`
	ClusterName    string   `yaml:"cluster_name"`
	AuthIPs        []string `yaml:"auth_ips"`
	BlackListIPs   []string `yaml:"black_list_ips"`
}

func (p *ProxyConfig) GetMasterNodefromClusterByName(clusterName string) *NodeConfig {
	if p == nil || p.Clusters == nil {
		fmt.Printf("GetMasterNodefromClusterByName p==nil or p.Clusters==nil")
		return nil
	}
	node := p.Clusters[clusterName]
	if node == nil || node.Master == nil {
		fmt.Printf("GetMasterNodefromClusterByName cluster %s do not exist", clusterName)
		return nil
	}
	return node.Master
}

func (p *ProxyConfig) GetSlaveNodesfromClusterByName(clusterName string) []*NodeConfig {
	if p == nil || p.Clusters == nil {
		fmt.Printf("GetSlaveNodesfromCluster p==nil or p.Clusters==nil")
		return nil
	}
	node := p.Clusters[clusterName]
	if node == nil {
		fmt.Printf("GetSlaveNodesfromCluster cluster %s do not exist", clusterName)
		return nil
	}
	return node.Slaves
}

func (p *ProxyConfig) GetUserByName(username string) *UserConfig {
	if p == nil || p.Users == nil {
		fmt.Printf("GetUserByName p==nil or p.Users==nil")
		return nil
	}
	user := p.Users[username]
	if user == nil {
		fmt.Printf("GetUserByName user %s do not exist", username)
		return nil
	}
	return user
}

func (c *CONF) parseConfigFile(proxyConfig *ProxyConfig) error {
	data, err := ioutil.ReadFile(c.path)
	if err == nil {
		err = yaml.Unmarshal([]byte(data), proxyConfig)
		if err == nil {
			if !validateConfig(proxyConfig) {
				err = fmt.Errorf("config is invalidate")
			}
		}

	}
	return err
}

func (c *CONF) GetConfig() *ProxyConfig {
	c.mu.RLock()
	proxyConfig := c.proxyConfig
	c.mu.RUnlock()
	return proxyConfig
}

func (c *CONF) CheckConfigUpdate() {
	if c.proxyConfig.Global.ConfAutoload == 1 {
		for {
			time.Sleep(time.Minute)
			fmt.Printf("CheckConfigUpdate checking")
			fileinfo, err := os.Stat(c.path)
			if err != nil {
				fmt.Printf("CheckConfigUpdate error %s", err.Error())
				continue
			}
			//配置文件发生修改
			if c.lastModifiedTime.Before(fileinfo.ModTime()) {
				fmt.Printf("CheckConfigUpdate config change and load new config")
				defaultProxyConfig := getDefaultProxyConfig()
				err = c.parseConfigFile(defaultProxyConfig)
				if err != nil {
					fmt.Printf("CheckConfigUpdate error %s", err.Error())
					continue
				}
				c.lastModifiedTime = fileinfo.ModTime()
				//单独goroutine执行加锁
				c.mu.Lock()
				c.proxyConfig = defaultProxyConfig
				c.mu.Unlock()
				fmt.Printf("CheckConfigUpdate new config load success")
			}
		}

	}
}

func LoadConfig(path string) (*CONF, error) {
	fileinfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	conf.path = path
	defaultProxyConfig := getDefaultProxyConfig()
	err = conf.parseConfigFile(defaultProxyConfig)
	if err != nil {
		return nil, err
	}
	conf.lastModifiedTime = fileinfo.ModTime()
	conf.proxyConfig = defaultProxyConfig
	return &conf, err
}

//检查配置文件的有效性
func validateConfig(cfg *ProxyConfig) bool {
	if cfg == nil {
		return false
	}

	if len(cfg.Clusters) == 0 {
		fmt.Printf("ValidateConfig 0 cluster")
		return false
	}

	if len(cfg.Users) == 0 {
		fmt.Printf("ValidateConfig 0 user")
		return false
	}

	for username, user := range cfg.Users {
		clusterName := user.ClusterName
		if _, ok := cfg.Clusters[clusterName]; !ok {
			fmt.Printf("ValidateConfig cluster %s belong to user %s do not exist", clusterName, username)
			return false
		}
	}

	for clusterName, cluster := range cfg.Clusters {
		if cluster.Master == nil {
			fmt.Printf("ValidateConfig cluster %s do not have master node", clusterName)
			return false
		}
	}

	return true
}

func getDefaultProxyConfig() *ProxyConfig {
	cfg := ProxyConfig{
		Global: &GlobalConfig{
			Port:              3306,
			ManagePort:        3307,
			MaxConnections:    2000,
			LogLevel:          1,
			LogFilename:       "./log/dbatman.log",
			LogMaxSize:        2014,
			ClientTimeout:     1800,
			ServerTimeout:     1800,
			WriteTimeInterval: 10,
			ConfAutoload:      1,
			AuthIPs:           []string{"127.0.0.1"},
		},
	}
	return &cfg
}
