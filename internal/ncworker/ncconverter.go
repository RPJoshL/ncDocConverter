package ncworker

type NcConverter struct {
	NextcloudBaseUrl	string`json:"nextcloudUrl"`
	Username			string`json:"username"`
	App
	SourceDir			string`json:"users"`
	DestinationDir		string`json:"users"`
}