package plugin

// Plugin can intercept Service Manager operations and
// augment them with additional logic.
// To intercept SM operations a plugin implements one or more of
// the interfaces defined in this package.
type Plugin interface{}
