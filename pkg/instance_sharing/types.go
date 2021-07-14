package instance_sharing

const (
	ReferencePlanDescription = "Allows to create a reference to a shared service instance from any environment in a subaccount and manage service bindings to that service instance."
	ReferencePlanName        = "reference-instance"

	SupportsInstanceSharingKey = "supportsInstanceSharing"

	ReferencedInstanceIDKey         = "referenced_instance_id"
	ReferencedInstanceIDTitle       = "Shared Instance ID "
	ReferencedInstanceIDDescription = "Specify the ID of the instance to which you want to create a reference."

	SelectorsKey           = "selectors"
	BySelectorsTitle       = "Shared Instance Attributes "
	BySelectorsDescription = "Find the shared instance to which you want to create a reference by specifying its various attributes, such as instance name, plan name, or labels."

	ReferenceInstanceNameSelectorKey         = "instance_name_selector"
	ReferenceInstanceNameSelectorTitle       = "Instance Name"
	ReferenceInstanceNameSelectorDescription = "Specify the instance name of the shared instance to which you want to create a reference."

	ReferencePlanNameSelectorKey         = "plan_name_selector"
	ReferencePlanNameSelectorTitle       = "Plan Name"
	ReferencePlanNameSelectorDescription = "Specify the plan name of the shared instance to which you want to create a reference."

	ReferenceLabelSelectorKey         = "instance_label_selector"
	ReferenceLabelSelectorTitle       = "Label"
	ReferenceLabelSelectorDescription = "Specify the labels to find the shared instance to which you want to create a reference. For example: “origin eq ‘eu’” returns an instance whose origin is in the EU. You can add multiple label queries to a single search."
)
