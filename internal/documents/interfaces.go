package documents

type Service interface {
	CreateDocument(title string, ownerId int) (*Document, error)
	GetDocument(documentId int) (*Document, error)
	GetUserDocuments(userId int) ([]Document, error)
	UpdateDocumentTitle(documentId int, title string) error
	DeleteDocument(documentId int) error
	GetDocumentEvents(documentId int, limit int) ([]Event, error)
	HasDocumentAccess(userId, documentId int) (bool, error)
}
