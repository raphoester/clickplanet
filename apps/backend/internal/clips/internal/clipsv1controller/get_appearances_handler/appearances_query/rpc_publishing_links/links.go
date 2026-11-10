// Where each post is, as the publishing module's daily snapshot read it: publishing.v1.InternalService/GetPostLinks.
package rpc_publishing_links

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler/appearances_query"
)

func New(client publishingv1connect.InternalServiceClient, logger *slog.Logger) Links {
	return Links{client: client, logger: logger}
}

type Links struct {
	client publishingv1connect.InternalServiceClient
	logger *slog.Logger
}

var _ appearances_query.Links = Links{}

// None when publishing does not answer: a player still reads the posts that name it, each linking to the channel.
func (l Links) Of(ctx context.Context, posts []string) (map[appearances_query.PostOnPlatform]string, error) {
	answer, err := l.client.GetPostLinks(ctx, connect.NewRequest(&publishingv1.GetPostLinksRequest{PostIds: posts}))
	if err != nil {
		l.logger.Warn("the posts' links are left out: publishing did not answer", slog.String("error", err.Error()))
		return map[appearances_query.PostOnPlatform]string{}, nil
	}
	links := make(map[appearances_query.PostOnPlatform]string, len(answer.Msg.GetLinks()))
	for _, link := range answer.Msg.GetLinks() {
		links[appearances_query.PostOnPlatform{Post: link.GetPostId(), Platform: link.GetPlatform()}] = link.GetUrl()
	}
	return links, nil
}
