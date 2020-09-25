package flexmgo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git.kanosolution.net/kano/dbflex"
	"github.com/eaciit/toolkit"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Connection struct {
	dbflex.ConnectionBase `bson:"-" json:"-"`
	ctx                   context.Context
	client                *mongo.Client
	db                    *mongo.Database
	sess                  mongo.Session
}

func (c *Connection) Connect() error {
	configString := "?"
	for k, v := range c.Config {
		configString += k + "=" + v.(string) + "&"
	}

	connURI := "mongodb://"

	// Check if credentials given	
	if c.User != "" && c.Password != "" {
		connURI += c.User + ":" + c.Password + "@"
	}

	connURI += c.Host + "/"
	connURI += configString

	opts := options.Client().ApplyURI(connURI)

	for k, v := range c.Config {
		klow := strings.ToLower(k)
		switch klow {
		case "serverselectiontimeout":
			opts.SetServerSelectionTimeout(
				time.Duration(toolkit.ToInt(v, toolkit.RoundingAuto)) * time.Millisecond)

		case "replicaset":
			opts.SetReplicaSet(v.(string))
			//opts.SetWriteConcern()
		}
	}

	//toolkit.Logger().Debugf("opts: %s", toolkit.JsonString(opts))
	client, err := mongo.NewClient(opts)
	if err != nil {
		return err
	}

	//toolkit.Logger().Debug("client generated: OK")
	if c.ctx == nil {
		c.ctx = context.Background()
	}

	//toolkit.Logger().Debug("context generated: OK")
	if err = client.Connect(c.ctx); err != nil {
		return err
	}

	//toolkit.Logger().Debug("client connected: OK")
	if err = client.Ping(c.ctx, nil); err != nil {
		return err
	}

	c.client = client
	if c.Database != "" {
		c.db = c.client.Database(c.Database)
	}

	return nil
}

func (c *Connection) Mdb() *mongo.Database {
	return c.db
}

func (c *Connection) State() string {
	if c.client == nil {
		return dbflex.StateUnknown
	} else {
		return dbflex.StateConnected
	}
}

func (c *Connection) Close() {
	if c.client != nil {
		c.client.Disconnect(c.ctx)
		c.client = nil
	}
}

func (c *Connection) NewQuery() dbflex.IQuery {
	q := new(Query)
	q.SetThis(q)
	q.SetConnection(c)

	return q
}

func (c *Connection) DropTable(name string) error {
	return c.db.Collection(name).Drop(c.ctx)
}

func (c *Connection) BeginTx() error {
	if c.sess != nil {
		return fmt.Errorf("session already exist. Pls commit or rollback last")
	}

	sess, err := c.client.StartSession()
	if err != nil {
		return fmt.Errorf("unable to start new transaction. %s", err.Error())
	}
	sess.StartTransaction()
	c.sess = sess
	return nil
}

func (c *Connection) Commit() error {
	if c.sess == nil {
		return fmt.Errorf("transaction session is not exists yet")
	}

	err := c.sess.CommitTransaction(c.ctx)
	if err != nil {
		return fmt.Errorf("unable to commit. %s", err.Error())
	}

	c.sess = nil
	return nil
}

func (c *Connection) Rollback() error {
	if c.sess == nil {
		return fmt.Errorf("transaction session is not exists yet")
	}

	err := c.sess.AbortTransaction(c.ctx)
	if err != nil {
		return fmt.Errorf("unable to commit. %s", err.Error())
	}

	c.sess = nil
	return nil
}

func (c *Connection) IsTx() bool {
	return c.sess != nil
}
