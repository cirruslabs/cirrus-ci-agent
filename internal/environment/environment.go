package environment

type Environment struct {
	env             map[string]string
	sensitiveValues []string
}

func New(items map[string]string) *Environment {
	env := NewEmpty()

	env.Merge(items)

	return env
}

func NewEmpty() *Environment {
	return &Environment{
		env:             map[string]string{},
		sensitiveValues: []string{},
	}
}

func (env *Environment) Get(key string) string {
	return env.env[key]
}

func (env *Environment) Lookup(key string) (string, bool) {
	value, ok := env.env[key]

	return value, ok
}

func (env *Environment) Set(key string, value string) {
	env.env[key] = value
}

func (env *Environment) Merge(otherEnv map[string]string) {
	if len(otherEnv) == 0 {
		return
	}

	// Accommodate new environment variables
	for key, value := range otherEnv {
		env.env[key] = value
	}

	// Do one more expansion pass since we've introduced
	// new and potentially unexpanded variables
	env.env = ExpandEnvironmentRecursively(env.env)
}

func (env *Environment) Items() map[string]string {
	return env.env
}

func (env *Environment) AddSensitiveValues(sensitiveValues ...string) {
	env.sensitiveValues = append(env.sensitiveValues, sensitiveValues...)
}

func (env *Environment) SensitiveValues() []string {
	return env.sensitiveValues
}
