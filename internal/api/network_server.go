package api

import (
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/brocaar/lora-app-server/api"
	"github.com/brocaar/lora-app-server/internal/api/auth"
	"github.com/brocaar/lora-app-server/internal/config"
	"github.com/brocaar/lora-app-server/internal/storage"
	"github.com/brocaar/loraserver/api/ns"
)

// NetworkServerAPI exports the NetworkServer related functions.
type NetworkServerAPI struct {
	validator auth.Validator
}

// NewNetworkServerAPI creates a new NetworkServerAPI.
func NewNetworkServerAPI(validator auth.Validator) *NetworkServerAPI {
	return &NetworkServerAPI{
		validator: validator,
	}
}

// Create creates the given network-server.
func (a *NetworkServerAPI) Create(ctx context.Context, req *pb.CreateNetworkServerRequest) (*pb.CreateNetworkServerResponse, error) {
	if err := a.validator.Validate(ctx,
		auth.ValidateNetworkServersAccess(auth.Create, 0),
	); err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "authentication failed: %s", err)
	}

	ns := storage.NetworkServer{
		Name:                        req.Name,
		Server:                      req.Server,
		CACert:                      req.CaCert,
		TLSCert:                     req.TlsCert,
		TLSKey:                      req.TlsKey,
		RoutingProfileCACert:        req.RoutingProfileCACert,
		RoutingProfileTLSCert:       req.RoutingProfileTLSCert,
		RoutingProfileTLSKey:        req.RoutingProfileTLSKey,
		GatewayDiscoveryEnabled:     req.GatewayDiscoveryEnabled,
		GatewayDiscoveryInterval:    int(req.GatewayDiscoveryInterval),
		GatewayDiscoveryTXFrequency: int(req.GatewayDiscoveryTXFrequency),
		GatewayDiscoveryDR:          int(req.GatewayDiscoveryDR),
	}

	err := storage.Transaction(config.C.PostgreSQL.DB, func(tx sqlx.Ext) error {
		return storage.CreateNetworkServer(tx, &ns)
	})
	if err != nil {
		return nil, errToRPCError(err)
	}

	return &pb.CreateNetworkServerResponse{
		Id: ns.ID,
	}, nil
}

// Get returns the network-server matching the given id.
func (a *NetworkServerAPI) Get(ctx context.Context, req *pb.GetNetworkServerRequest) (*pb.GetNetworkServerResponse, error) {
	if err := a.validator.Validate(ctx,
		auth.ValidateNetworkServerAccess(auth.Read, req.Id),
	); err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "authentication failed: %s", err)
	}

	n, err := storage.GetNetworkServer(config.C.PostgreSQL.DB, req.Id)
	if err != nil {
		return nil, errToRPCError(err)
	}

	var region string
	var version string

	nsClient, err := config.C.NetworkServer.Pool.Get(n.Server, []byte(n.CACert), []byte(n.TLSCert), []byte(n.TLSKey))
	if err == nil {
		resp, err := nsClient.GetVersion(context.Background(), &ns.GetVersionRequest{})
		if err == nil {
			region = resp.Region.String()
			version = resp.Version
		}
	}

	return &pb.GetNetworkServerResponse{
		Id:                          n.ID,
		CreatedAt:                   n.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:                   n.UpdatedAt.Format(time.RFC3339Nano),
		Name:                        n.Name,
		Server:                      n.Server,
		CaCert:                      n.CACert,
		TlsCert:                     n.TLSCert,
		RoutingProfileCACert:        n.RoutingProfileCACert,
		RoutingProfileTLSCert:       n.RoutingProfileTLSCert,
		GatewayDiscoveryEnabled:     n.GatewayDiscoveryEnabled,
		GatewayDiscoveryInterval:    uint32(n.GatewayDiscoveryInterval),
		GatewayDiscoveryTXFrequency: uint32(n.GatewayDiscoveryTXFrequency),
		GatewayDiscoveryDR:          uint32(n.GatewayDiscoveryDR),
		Region:                      region,
		Version:                     version,
	}, nil
}

// Update updates the given network-server.
func (a *NetworkServerAPI) Update(ctx context.Context, req *pb.UpdateNetworkServerRequest) (*pb.UpdateNetworkServerResponse, error) {
	if err := a.validator.Validate(ctx,
		auth.ValidateNetworkServerAccess(auth.Update, req.Id),
	); err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "authentication failed: %s", err)
	}

	ns, err := storage.GetNetworkServer(config.C.PostgreSQL.DB, req.Id)
	if err != nil {
		return nil, errToRPCError(err)
	}

	ns.Name = req.Name
	ns.Server = req.Server
	ns.CACert = req.CaCert
	ns.TLSCert = req.TlsCert
	ns.RoutingProfileCACert = req.RoutingProfileCACert
	ns.RoutingProfileTLSCert = req.RoutingProfileTLSCert
	ns.GatewayDiscoveryEnabled = req.GatewayDiscoveryEnabled
	ns.GatewayDiscoveryInterval = int(req.GatewayDiscoveryInterval)
	ns.GatewayDiscoveryTXFrequency = int(req.GatewayDiscoveryTXFrequency)
	ns.GatewayDiscoveryDR = int(req.GatewayDiscoveryDR)

	if req.TlsKey != "" {
		ns.TLSKey = req.TlsKey
	}
	if ns.TLSCert == "" {
		ns.TLSKey = ""
	}

	if req.RoutingProfileTLSKey != "" {
		ns.RoutingProfileTLSKey = req.RoutingProfileTLSKey
	}
	if ns.RoutingProfileTLSCert == "" {
		ns.RoutingProfileTLSKey = ""
	}

	err = storage.Transaction(config.C.PostgreSQL.DB, func(tx sqlx.Ext) error {
		return storage.UpdateNetworkServer(tx, &ns)
	})
	if err != nil {
		return nil, errToRPCError(err)
	}

	return &pb.UpdateNetworkServerResponse{}, nil
}

// Delete deletes the network-server matching the given id.
func (a *NetworkServerAPI) Delete(ctx context.Context, req *pb.DeleteNetworkServerRequest) (*pb.DeleteNetworkServerResponse, error) {
	if err := a.validator.Validate(ctx,
		auth.ValidateNetworkServerAccess(auth.Delete, req.Id),
	); err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "authentication failed: %s", err)
	}

	err := storage.Transaction(config.C.PostgreSQL.DB, func(tx sqlx.Ext) error {
		return storage.DeleteNetworkServer(tx, req.Id)
	})
	if err != nil {
		return nil, errToRPCError(err)
	}

	return &pb.DeleteNetworkServerResponse{}, nil
}

// List lists the available network-servers.
func (a *NetworkServerAPI) List(ctx context.Context, req *pb.ListNetworkServerRequest) (*pb.ListNetworkServerResponse, error) {
	if err := a.validator.Validate(ctx,
		auth.ValidateNetworkServersAccess(auth.List, req.OrganizationID),
	); err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "authentication failed: %s", err)
	}

	isAdmin, err := a.validator.GetIsAdmin(ctx)
	if err != nil {
		return nil, errToRPCError(err)
	}

	var count int
	var nss []storage.NetworkServer

	if req.OrganizationID == 0 {
		if isAdmin {
			count, err = storage.GetNetworkServerCount(config.C.PostgreSQL.DB)
			if err != nil {
				return nil, errToRPCError(err)
			}
			nss, err = storage.GetNetworkServers(config.C.PostgreSQL.DB, int(req.Limit), int(req.Offset))
			if err != nil {
				return nil, errToRPCError(err)
			}
		}
	} else {
		count, err = storage.GetNetworkServerCountForOrganizationID(config.C.PostgreSQL.DB, req.OrganizationID)
		if err != nil {
			return nil, errToRPCError(err)
		}
		nss, err = storage.GetNetworkServersForOrganizationID(config.C.PostgreSQL.DB, req.OrganizationID, int(req.Limit), int(req.Offset))
		if err != nil {
			return nil, errToRPCError(err)
		}
	}

	resp := pb.ListNetworkServerResponse{
		TotalCount: int64(count),
	}

	for _, ns := range nss {
		resp.Result = append(resp.Result, &pb.GetNetworkServerResponse{
			Id:        ns.ID,
			CreatedAt: ns.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt: ns.UpdatedAt.Format(time.RFC3339Nano),
			Name:      ns.Name,
			Server:    ns.Server,
		})
	}

	return &resp, nil
}
