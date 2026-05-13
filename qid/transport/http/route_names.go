package http

type RouteName int

//go:generate go tool enumer -type=RouteName -transform=snake -output=route_names_string.go

const (
	UserLogin RouteName = iota

	CreateUser
	ListUsers

	CreateTeam
	ListTeams
	GetTeam
	UpdateTeam
	DeleteTeam

	CreateTeamMember
	UpdateTeamMember
	DeleteTeamMember

	CreatePipeline
	UpdatePipeline
	GetPipeline
	DeletePipeline
	ListPipelines

	GetPipelineImage
	CreatePipelineImage

	TriggerPipelineJob
	GetPipelineJob

	CreateJobBuild
	UpdateJobBuild
	DeleteJobBuild
	ListJobBuilds

	GetPipelineResource
	UpdatePipelineResource
	TriggerPipelineResource
	CreateResourceVersion
	ListResourceVersions
)
