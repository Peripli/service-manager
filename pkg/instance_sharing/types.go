package instance_sharing

const (
	ReferencePlanDescription = "Allows to create a reference to a shared service instance from any environment in a subaccount and manage service bindings to that service instance."
	ReferencePlanName        = "reference-instance"

	SupportsInstanceSharingKey = "supportsInstanceSharing"

	ReferencedInstanceIDKey         = "referenced_instance_id"
	ReferencedInstanceIDTitle       = "Shared Instance ID"
	ReferencedInstanceIDDescription = "Specify the ID of the instance to which you want to create the reference."

	SelectorsKey           = "selectors"
	BySelectorsTitle       = "Find by"
	BySelectorsDescription = "You can create a reference to a shared service instance without providing its ID. Use instead various attributes, such as plan, instance name, or labels to find a matching shared instance to which to create the reference."

	ReferenceInstanceNameSelectorKey         = "instance_name_selector"
	ReferenceInstanceNameSelectorTitle       = "Instance Name"
	ReferenceInstanceNameSelectorDescription = "Specify the instance name of the shared instance to which you want to create the reference."

	ReferencePlanNameSelectorKey         = "plan_name_selector"
	ReferencePlanNameSelectorTitle       = "Plan Name"
	ReferencePlanNameSelectorDescription = "Specify the plan name of the shared instance to which you want to create the reference."

	ReferenceLabelSelectorKey         = "instance_label_selector"
	ReferenceLabelSelectorTitle       = "Labels"
	ReferenceLabelSelectorDescription = "Use label query to find the shared instance to which you want to create the reference. For example: “origin eq ‘eu’” returns an instance whose origin is in the EU. You can add multiple label queries to a single search."
)
