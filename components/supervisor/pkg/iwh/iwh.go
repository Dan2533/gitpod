// Copyright (c) 2020 TypeFox GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package iwh

import (
	"context"
	"os"
	"time"

	daemon "github.com/gitpod-io/gitpod/ws-daemon/api"
	"golang.org/x/xerrors"

	"google.golang.org/grpc"
)

// NewInWorkspaceHelper creates a new in-workspace helper client
func NewInWorkspaceHelper(ctx context.Context) (daemon.InWorkspaceHelperClient, *grpc.ClientConn, error) {
	const socketFN = "/.workspace/daemon.sock"

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		if _, err := os.Stat(socketFN); err == nil {
			break
		}

		select {
		case <-t.C:
			continue
		case <-ctx.Done():
			return nil, nil, xerrors.Errorf("socket did not appear before context was canceled")
		}
	}

	conn, err := grpc.DialContext(ctx, "unix://"+socketFN, grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	return daemon.NewInWorkspaceHelperClient(conn), conn, nil
}

// func NewInWorkspaceHelper(checkoutLocation string, pauseTheia chan<- bool) *InWorkspaceHelper {
// 	return &InWorkspaceHelper{
// 		backup: &backupService{
// 			checkoutLocation: checkoutLocation,
// 			contentReadyChan: make(chan struct{}),
// 			pauseChan:        pauseTheia,
// 		},
// 		teardown: &teardownService{
// 			cond:          sync.NewCond(&sync.Mutex{}),
// 			triggerBackup: make(chan chan<- bool),
// 		},
// 		idmapper: &idmapperService{
// 			cond:          sync.NewCond(&sync.Mutex{}),
// 			triggerUIDMap: make(chan *triggerNewuidmapReq),
// 		},
// 	}
// }

// // InWorkspaceHelper provides services backed by ws-daemon canaries
// type InWorkspaceHelper struct {
// 	teardown *teardownService
// 	backup   *backupService
// 	idmapper *idmapperService
// }

// // RegisterGRPC registers a gRPC service
// func (iwh *InWorkspaceHelper) RegisterGRPC(srv *grpc.Server) {
// 	type grpcIWH struct {
// 		*teardownIWH
// 		*backupIWH
// 		*idmapperIWH
// 	}
// 	daemon.RegisterInWorkspaceHelperServer(srv, grpcIWH{
// 		teardownIWH: &teardownIWH{iwh.teardown},
// 		backupIWH:   &backupIWH{iwh.backup},
// 		idmapperIWH: &idmapperIWH{iwh.idmapper},
// 	})
// }

// // ContentState provides access to the workspace's content state
// func (iwh *InWorkspaceHelper) ContentState() ContentState {
// 	return iwh.backup
// }

// // TeardownService provides access to the canary-backed backup service
// func (iwh *InWorkspaceHelper) TeardownService() TeardownService {
// 	return iwh.teardown
// }

// // IDMapperService provides access to the canary-backed id mapper service
// func (iwh *InWorkspaceHelper) IDMapperService() IDMapperService {
// 	return iwh.idmapper
// }

// // ContentState signals the workspace content state
// type ContentState interface {
// 	MarkContentReady(src csapi.WorkspaceInitSource)
// 	ContentReady() <-chan struct{}
// 	ContentSource() (src csapi.WorkspaceInitSource, ok bool)
// }

// // TeardownService is the supervisor-facing, canary backed, backup service
// type TeardownService interface {
// 	Available() <-chan struct{}
// 	Teardown(ctx context.Context) error
// }

// var _ ContentState = &backupService{}
// var _ TeardownService = &teardownService{}

// type backupService struct {
// 	checkoutLocation string
// 	pauseChan        chan<- bool

// 	contentReadyChan chan struct{}
// 	contentSource    csapi.WorkspaceInitSource
// }

// // MarkContentReady marks the workspace content as available.
// // This function is not synchronized and must be called from a single thread/go routine only.
// func (iwh *backupService) MarkContentReady(src csapi.WorkspaceInitSource) {
// 	iwh.contentSource = src
// 	close(iwh.contentReadyChan)
// }

// // ContentReady returns a chan that closes when the content becomes available
// func (iwh *backupService) ContentReady() <-chan struct{} {
// 	return iwh.contentReadyChan
// }

// // ContentSource returns the init source of the workspace content.
// // The value returned here is only OK after ContentReady() was closed.
// func (iwh *backupService) ContentSource() (src csapi.WorkspaceInitSource, ok bool) {
// 	select {
// 	case <-iwh.contentReadyChan:
// 	default:
// 		return "", false
// 	}
// 	return iwh.contentSource, true
// }

// type teardownService struct {
// 	triggerBackup   chan chan<- bool
// 	cond            *sync.Cond
// 	canaryAvailable int32
// }

// // Teardown prepares a workspace content backup and unmounts the shiftfs mark
// func (iwh *teardownService) Teardown(ctx context.Context) error {
// 	rc := make(chan bool)

// 	select {
// 	case iwh.triggerBackup <- rc:
// 	case <-ctx.Done():
// 		return status.Error(codes.DeadlineExceeded, ctx.Err().Error())
// 	default:
// 		return status.Error(codes.Unavailable, "no teardown canary available")
// 	}

// 	select {
// 	case success := <-rc:
// 		if !success {
// 			return status.Error(codes.Internal, "teardown failed")
// 		}
// 	case <-ctx.Done():
// 		return status.Error(codes.DeadlineExceeded, ctx.Err().Error())
// 	}

// 	return nil
// }

// // CanaryAvailable returns true if there's a backup canary available
// func (iwh *teardownService) Available() <-chan struct{} {
// 	res := make(chan struct{})
// 	go func() {
// 		iwh.cond.L.Lock()
// 		defer iwh.cond.L.Unlock()

// 		for iwh.canaryAvailable <= 0 {
// 			iwh.cond.Wait()
// 		}

// 		close(res)
// 	}()

// 	return res
// }

// type teardownIWH struct {
// 	*teardownService
// }

// // BackupCanary can prepare workspace content backups. The canary is supposed to be triggered
// // when the workspace is about to shut down, e.g. using the PreStop hook of a Kubernetes container.
// func (iwh *teardownIWH) TeardownCanary(srv daemon.InWorkspaceHelper_TeardownCanaryServer) error {
// 	iwh.cond.L.Lock()
// 	iwh.canaryAvailable++
// 	iwh.cond.Broadcast()
// 	iwh.cond.L.Unlock()

// 	defer func() {
// 		iwh.cond.L.Lock()
// 		iwh.canaryAvailable--
// 		iwh.cond.Broadcast()
// 		iwh.cond.L.Unlock()
// 	}()

// 	rc := <-iwh.triggerBackup
// 	if rc == nil {
// 		return status.Error(codes.FailedPrecondition, "trigger chan closed")
// 	}

// 	err := srv.Send(&daemon.TeardownRequest{})
// 	if err != nil {
// 		return err
// 	}

// 	req, err := srv.Recv()
// 	if err != nil {
// 		log.WithError(err).Error("backup prep failed")
// 		rc <- false
// 	}
// 	rc <- req.Success

// 	return nil
// }

// type backupIWH struct {
// 	*backupService
// }

// // PauseTheia can pause the Theia process and all its children. As long as the request stream
// // is held Theia will be paused.
// // This is a stop-the-world mechanism for preventing concurrent modification during backup.
// func (iwh *backupIWH) PauseTheia(srv daemon.InWorkspaceHelper_PauseTheiaServer) error {
// 	iwh.pauseChan <- true
// 	defer func() {
// 		iwh.pauseChan <- false
// 	}()

// 	_, err := srv.Recv()
// 	if err != nil {
// 		return err
// 	}

// 	return srv.SendAndClose(&daemon.PauseTheiaResponse{})
// }

// const (
// 	// maxPendingChanges is the limit beyond which we no longer report pending changes.
// 	// For example, if a workspace has then 150 untracked files, we'll report the first
// 	// 100 followed by "... and 50 more".
// 	//
// 	// We do this to keep the load on our infrastructure light and because beyond this number
// 	// the changes are irrelevant anyways.
// 	maxPendingChanges = 100
// )

// // GitStatus provides the current state of the main Git repo at the workspace's checkout location
// func (iwh *backupIWH) GitStatus(ctx context.Context, req *daemon.GitStatusRequest) (*daemon.GitStatusResponse, error) {
// 	//
// 	// BEWARE
// 	// This functionality is duplicated in ws-daemon.
// 	// When we make the backup work without the PLIS we should de-duplicate this implementation.
// 	//

// 	cl := &git.Client{Location: iwh.checkoutLocation}
// 	s, err := cl.Status(ctx)
// 	if err != nil {
// 		return nil, status.Error(codes.Internal, err.Error())
// 	}

// 	res := s.ToAPI()
// 	return &daemon.GitStatusResponse{Repo: res}, nil
// }

// // IDMapperService is the supervisor-facing, canary-backed UID/GID mapping service for user namespace support
// type IDMapperService interface {
// 	Available() <-chan struct{}
// 	WriteIDMap(ctx context.Context, req *daemon.UidmapCanaryRequest) error
// }

// type triggerNewuidmapReq struct {
// 	Req  *daemon.UidmapCanaryRequest
// 	Resp chan error
// }

// var _ IDMapperService = &idmapperService{}

// type idmapperService struct {
// 	canaryAvailable int32
// 	cond            *sync.Cond
// 	triggerUIDMap   chan *triggerNewuidmapReq
// }

// // CanaryAvailable returns true if there's a canaray available.
// // If there isn't, calling Newuidmap or Newguidmap won't succeed.
// func (iwh *idmapperService) Available() <-chan struct{} {
// 	res := make(chan struct{})
// 	go func() {
// 		iwh.cond.L.Lock()
// 		defer iwh.cond.L.Unlock()

// 		for iwh.canaryAvailable <= 0 {
// 			iwh.cond.Wait()
// 		}

// 		close(res)
// 	}()

// 	return res
// }

// // WriteIDMap asks the canary to create a new uidmap. If there's no canary
// // available, this function will block until one becomes available or the
// // context is canceled.
// func (iwh *idmapperService) WriteIDMap(ctx context.Context, req *daemon.UidmapCanaryRequest) error {
// 	iwh.cond.L.Lock()
// 	avail := iwh.canaryAvailable > 0
// 	iwh.cond.L.Unlock()
// 	if !avail {
// 		return status.Error(codes.Unavailable, "not available")
// 	}

// 	trigger := &triggerNewuidmapReq{
// 		Req:  req,
// 		Resp: make(chan error, 1),
// 	}

// 	select {
// 	case iwh.triggerUIDMap <- trigger:
// 	case <-ctx.Done():
// 		return ctx.Err()
// 	}

// 	select {
// 	case err := <-trigger.Resp:
// 		return err
// 	case <-ctx.Done():
// 		return ctx.Err()
// 	}
// }

// type idmapperIWH struct {
// 	*idmapperService
// }

// func (iwh *idmapperIWH) UidmapCanary(srv daemon.InWorkspaceHelper_UidmapCanaryServer) error {
// 	iwh.cond.L.Lock()
// 	iwh.canaryAvailable++
// 	iwh.cond.Broadcast()
// 	iwh.cond.L.Unlock()

// 	defer func() {
// 		iwh.cond.L.Lock()
// 		iwh.canaryAvailable--
// 		iwh.cond.Broadcast()
// 		iwh.cond.L.Unlock()
// 	}()

// 	for {
// 		select {
// 		case req := <-iwh.triggerUIDMap:
// 			err := srv.Send(req.Req)
// 			if err != nil {
// 				req.Resp <- err
// 			}
// 			resp, err := srv.Recv()
// 			if err != nil {
// 				req.Resp <- err
// 			}
// 			if resp.ErrorCode > 0 {
// 				req.Resp <- status.Error(codes.Code(resp.ErrorCode), resp.Message)
// 			}

// 			req.Resp <- nil
// 		case <-srv.Context().Done():
// 			// canary dropped out - we're done here
// 		}
// 	}
// }
