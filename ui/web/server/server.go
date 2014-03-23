// Copyright 2013 Andreas Koch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"github.com/andreaskoch/allmark2/common/config"
	"github.com/andreaskoch/allmark2/common/index"
	"github.com/andreaskoch/allmark2/common/logger"
	"github.com/andreaskoch/allmark2/common/paths"
	"github.com/andreaskoch/allmark2/common/paths/webpaths"
	"github.com/andreaskoch/allmark2/services/conversion"
	"github.com/andreaskoch/allmark2/ui/web/server/handler"
	"github.com/gorilla/mux"
	"math"
	"net/http"
)

const (
	// Dynamic Routes
	ItemHandlerRoute       = "/{path:.*}"
	TagmapHandlerRoute     = "/tags.html"
	SitemapHandlerRoute    = "/sitemap.html"
	XmlSitemapHandlerRoute = "/sitemap.xml"
	RssHandlerRoute        = "/feed.rss"
	RobotsTxtHandlerRoute  = "/robots.txt"
	DebugHandlerRoute      = "/debug/index"
	SearchHandlerRoute     = "/search"
	WebSocketHandlerRoute  = "/ws"

	// Static Routes
	ThemeFolderRoute = "/theme/"
)

func New(logger logger.Logger, config *config.Config, converter conversion.Converter, itemIndex *index.ItemIndex) (*Server, error) {

	// file index
	fileIndex := index.CreateFileIndex(logger)

	// serve theme files
	baseFolder := config.MetaDataFolder()
	themeFolder := config.ThemeFolder()
	fileIndex.AddFolder(baseFolder, themeFolder)

	// pather factory
	patherFactory := webpaths.NewFactory(logger, itemIndex)

	return &Server{
		config:        config,
		logger:        logger,
		patherFactory: patherFactory,
		converter:     converter,
		itemIndex:     itemIndex,
		fileIndex:     fileIndex,
	}, nil

}

type Server struct {
	isRunning bool

	config        *config.Config
	logger        logger.Logger
	converter     conversion.Converter
	itemIndex     *index.ItemIndex
	fileIndex     *index.FileIndex
	patherFactory paths.PatherFactory
}

func (server *Server) IsRunning() bool {
	return server.isRunning
}

func (server *Server) Start() chan error {
	result := make(chan error)

	go func() {
		server.isRunning = true

		// register requst routers
		requestRouter := mux.NewRouter()
		requestRouter.HandleFunc(RobotsTxtHandlerRoute, handler.NewRobotsTxtHandler(server.logger, server.config, server.itemIndex, server.patherFactory).Func())
		requestRouter.HandleFunc(XmlSitemapHandlerRoute, handler.NewXmlSitemapHandler(server.logger, server.config, server.itemIndex, server.patherFactory).Func())
		requestRouter.HandleFunc(TagmapHandlerRoute, handler.NewTagsHandler(server.logger, server.config, server.itemIndex, server.patherFactory).Func())
		requestRouter.HandleFunc(SitemapHandlerRoute, handler.NewSitemapHandler(server.logger, server.config, server.itemIndex, server.patherFactory).Func())
		requestRouter.HandleFunc(DebugHandlerRoute, handler.NewDebugHandler(server.logger, server.itemIndex, server.fileIndex).Func())
		requestRouter.HandleFunc(RssHandlerRoute, handler.NewRssHandler(server.logger, server.config, server.itemIndex, server.patherFactory, server.converter).Func())
		requestRouter.HandleFunc(SearchHandlerRoute, handler.NewSearchHandler(server.logger, server.config, server.itemIndex, server.patherFactory).Func())
		requestRouter.HandleFunc(ItemHandlerRoute, handler.NewItemHandler(server.logger, server.config, server.itemIndex, server.fileIndex, server.patherFactory, server.converter).Func())

		// start http server: http
		httpBinding := server.getHttpBinding()
		server.logger.Info("Starting http server %q\n", httpBinding)

		if err := http.ListenAndServe(httpBinding, requestRouter); err != nil {
			result <- fmt.Errorf("Server failed with error: %v", err)
		} else {
			result <- nil
		}

		server.isRunning = false
	}()

	return result
}

func (server *Server) getHttpBinding() string {

	// validate the port
	port := server.config.Server.Http.Port
	if port < 1 || port > math.MaxUint16 {
		panic(fmt.Sprintf("%q is an invalid value for a port. Ports can only be in the range of %v to %v,", port, 1, math.MaxUint16))
	}

	return fmt.Sprintf(":%v", port)
}
