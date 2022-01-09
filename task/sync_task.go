package task

import (
	"github.com/gogf/gf/database/gdb"
	"github.com/gogf/gf/encoding/ghash"
	"github.com/gogf/gf/encoding/gjson"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/os/glog"
	"github.com/gogf/gf/os/gtime"
	"github.com/gorilla/feeds"
	"github.com/olivere/elastic/v7"
	"rsshub-auto-refresh/component"
	"rsshub-auto-refresh/model"
	"strconv"
)

func CallRSSApi(address, route string) (err error) {

	apiUrl := "http://localhost" + address + route
	if _, err = component.GetHttpClient().SetHeaderMap(getHeaders()).Get(apiUrl); err != nil {
		glog.Line().Println("Feed refresh cron job error : ", err)
	}

	return err
}

func StoreFeed(feed, tag, rsshubLink string) (err error) {
	var (
		feedObj  *feeds.Feed
		tagArray []string
	)
	feedObj = new(feeds.Feed)
	tagArray = make([]string, 0)
	_ = gjson.DecodeTo(feed, feedObj)
	_ = gjson.DecodeTo(tag, tagArray)
	err = addFeedChannelAndItem(feedObj, tagArray, rsshubLink)
	if err != nil {
		glog.Line().Println(err)
	}
	return err
}

func addFeedChannelAndItem(feed *feeds.Feed, tagList []string, rsshubLink string) error {

	feedID := strconv.FormatUint(ghash.RSHash64([]byte(feed.Link.Href+feed.Title)), 32)
	feedChannelModel := model.RssFeedChannel{
		Id:          feedID,
		Title:       feed.Title,
		ChannelDesc: feed.Description,
		ImageUrl:    feed.Image.Url,
		Link:        feed.Link.Href,
		RsshubLink:  rsshubLink,
	}

	feedItemModeList := make([]model.RssFeedItem, 0)
	for _, item := range feed.Items {
		feedItem := model.RssFeedItem{
			ChannelId:   feedID,
			Title:       item.Title,
			ChannelDesc: item.Description,
			Link:        item.Link.Href,
			Date:        gtime.New(item.Created.String()),
			Author:      item.Author.Name,
			InputDate:   gtime.Now(),
			Thumbnail:   item.Enclosure.Url,
		}
		uniString := feedItem.Link + feedItem.Title
		feedItemID := strconv.FormatUint(ghash.RSHash64([]byte(uniString)), 32)
		feedItem.Id = feedItemID
		feedItemModeList = append(feedItemModeList, feedItem)
	}

	tagModeList := make([]model.RssFeedTag, 0)
	for _, tagStr := range tagList {
		if tagStr == "" {
			continue
		}
		tagModel := model.RssFeedTag{
			Name:      tagStr,
			ChannelId: feedID,
			Title:     feed.Title,
			Date:      gtime.Now(),
		}

		tagModeList = append(tagModeList, tagModel)
	}

	err := component.GetDatabase().Transaction(func(tx *gdb.TX) error {
		var err error

		_, _ = tx.Save("rss_feed_channel", feedChannelModel)
		_, _ = tx.BatchInsertIgnore("rss_feed_tag", tagModeList)
		_, err = tx.BatchInsertIgnore("rss_feed_item", feedItemModeList)

		return err
	})
	if err != nil {
		g.Log().Error("insert rss feed data failed : ", err)
	}

	bulkRequest := component.GetESClient().Bulk()
	for _, feedItem := range feedItemModeList {
		esFeedItem := model.RssFeedItemESData{
			Id:              feedItem.Id,
			ChannelId:       feedItem.ChannelId,
			Title:           feedItem.Title,
			ChannelDesc:     feedItem.ChannelDesc,
			Thumbnail:       feedItem.Thumbnail,
			Link:            feedItem.Link,
			Date:            feedItem.Date,
			Author:          feedItem.Author,
			InputDate:       feedItem.InputDate,
			ChannelImageUrl: feedChannelModel.ImageUrl,
			ChannelTitle:    feedChannelModel.Title,
			ChannelLink:     feedChannelModel.Link,
		}
		indexReq := elastic.NewBulkIndexRequest().Index("rss_item").Id(feedItem.Id).Doc(esFeedItem)
		bulkRequest.Add(indexReq)
	}
	resp, err := bulkRequest.Do(component.GetESContext())
	if err != nil || resp.Errors {
		respStr := gjson.New(resp)
		g.Log().Errorf("bulk index request failed\nError message : %s \nResponse : %s", err, respStr)
	}

	return err
}

func getHeaders() map[string]string {
	headers := make(map[string]string)
	headers["accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"
	headers["user-agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.135 Safari/537.36 Edg/84.0.522.63"
	return headers
}
