package entities

func (FashionItem) TableName() string {
	return "fashion_items"
}

func (Category) TableName() string {
	return "categories"
}

func (ConversationalContext) TableName() string {
	return "conversational_contexts"
}

func (Message) TableName() string {
	return "messages"
}
