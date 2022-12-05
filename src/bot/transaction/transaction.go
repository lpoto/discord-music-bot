package transaction

import (
	"discord-music-bot/builder"
	"discord-music-bot/datastore"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type Transactions struct {
	id               uint
	log              *log.Logger
	session          func() *discordgo.Session
	interactions     map[string]chan *discordgo.Interaction
	interactionsSync sync.RWMutex
	datastore        *datastore.Datastore
	builder          *builder.Builder
	ready            func() bool
}

type Transaction struct {
	id              uint
	t               string
	interaction     *discordgo.Interaction
	guildID         string
	done            bool
	allTransactions *Transactions
	quiet           bool
}

// NewTransactions constructs a new object that handles the
// creation and holds data for Transaction objects
func NewTransactions(s func() *discordgo.Session, log *log.Logger, ds *datastore.Datastore, b *builder.Builder, ready func() bool) *Transactions {
	return &Transactions{
		id:           0,
		log:          log,
		interactions: make(map[string]chan *discordgo.Interaction),
		session:      s,
		datastore:    ds,
		builder:      b,
		ready:        ready,
	}
}

// New constructs a new Transaction objects.
// A transaction holds info about an event recieved to the
// bot from the user and then tries to optimize updating
// the music queues.
// NOTE: this creates a worker that waits the lifetime of
// the provided interaction (if not nil) and then defers it
// if it has not yet been responded to.
// NOTE: interactions are added to a channel and are used when updating
// as updating from interactions is much faster and does not block discord.
// This is useful as event defered interactions may be sometimes used for updating,
// but it should be noted that it is not known from which interaction the
// queue will be updated when calling transaction.UpdateQueue.
func (t *Transactions) New(tp string, guildID string, interaction *discordgo.Interaction) *Transaction {
	id := t.id
	t.id = (t.id + 1) % 100000
	t.log.WithFields(log.Fields{
		"ID":      id,
		"Type":    tp,
		"GuildID": guildID,
	}).Debug("Started new transaction ...")
	if interaction != nil {
		go t.addInteraction(interaction)
	}
	return &Transaction{
		t:               tp,
		id:              id,
		allTransactions: t,
		guildID:         guildID,
		interaction:     interaction,
		quiet:           false,
	}
}

// Interaction returns the interaction stored in the transaction
func (t *Transaction) Interaction() *discordgo.Interaction {
	return t.interaction
}

// Interaction returns the guildID
func (t *Transaction) GuildID() string {
	return t.guildID
}

// Refresh  marks the transaction as not yet completed.
// A transaction is automatically marked as completed when it
// is defered or when it updates a queue.
func (t *Transaction) Refresh() {
	t.allTransactions.log.WithFields(log.Fields{
		"ID":      t.id,
		"Type":    t.t,
		"GuildID": t.GuildID(),
	}).Debug(
		"Transaction refreshed",
	)
	t.done = false
}

// Defer marks the transaction as completed
// without updating the queue.
func (t *Transaction) Defer() {
	if t.done {
		return
	}
	t.allTransactions.log.WithFields(log.Fields{
		"ID":      t.id,
		"Type":    t.t,
		"GuildID": t.GuildID(),
	}).Debug(
		"Transaction done (defered)",
	)
	t.done = true
}

// UpdateQueue updates the queue after the provided timeout.
// This first tries to update the queue from the interactions stored
// in the Transactions object when new transactions are added.
// If unsuccessful, it updates it based on it's messageID and channelID.
func (t *Transaction) UpdateQueue(timeout time.Duration) error {
	if t.done {
		return nil
	}
	if !t.quiet {
		t.allTransactions.log.WithFields(log.Fields{
			"ID":      t.id,
			"Type":    t.t,
			"GuildID": t.GuildID(),
		}).Debug(
			"Transaction updating queue ...",
		)
	}
	defer func() {
		// NOTE: mark interaction as done
		// once the update is complete
		t.done = true
	}()

	clientID := t.allTransactions.session().State.User.ID
	guildID := t.guildID

	// NOTE: first fetch the queue as the queue message
	// is built from the queue object
	queue, err := t.allTransactions.datastore.Queue().GetQueue(
		clientID,
		guildID,
	)
	if err != nil {
		if !t.quiet {
			t.allTransactions.log.WithFields(log.Fields{
				"ID":      t.id,
				"Type":    t.t,
				"GuildID": t.GuildID(),
			}).Debugf(
				"Transaction queue updating failed: %v",
				err,
			)
		}
		return err
	}
	queue, err = t.allTransactions.datastore.Song().UpdateQueueWithSongs(
		queue,
	)
	if err != nil {
		if !t.quiet {
			t.allTransactions.log.WithFields(log.Fields{
				"ID":      t.id,
				"Type":    t.t,
				"GuildID": t.GuildID(),
			}).Debugf(
				"Transaction queue updating failed: %v",
				err,
			)
		}
		return err
	}
	// NOTE: get queue message's components based on the state of
	// the bot. If bot._ready = false, offline components will be added which
	// consist of a single disabled button.
	// If bot is not in voice channel in the queue's guild, inactive
	// components will be added which consist of Join button.
	// Otherwise all other buttons will be added.
	var c []discordgo.MessageComponent
	if !t.allTransactions.ready() {
		c = t.allTransactions.builder.Queue().GetOfflineQueueComponents(
			queue,
		)
	} else {
		vc, ok := t.allTransactions.session().VoiceConnections[guildID]
		if ok && vc.Ready {
			c = t.allTransactions.builder.Queue().GetMusicQueueComponents(
				queue,
			)
		} else {
			c = t.allTransactions.builder.Queue().GetInactiveQueueComponents(
				queue,
			)

		}
	}
	embed := t.allTransactions.builder.Queue().MapQueueToEmbed(queue)

	err = nil

	// NOTE: get the interactions buffer, and try to update the
	// queue from one of the stored interactions as it is much
	// faster and discord gives no limit on it.
	// If all these interaction responses fail, update
	// based on the queue's messageID and guildID
	t.allTransactions.interactionsSync.RLock()
	buffer, ok := t.allTransactions.interactions[guildID]
	t.allTransactions.interactionsSync.RUnlock()

	if !ok {
		buffer = make(chan *discordgo.Interaction)
	}
	fields := log.Fields{
		"ID":      t.id,
		"Type":    t.t,
		"GuildID": t.GuildID(),
	}
updateLoop:
	for {
		select {
		case i := <-buffer:
			err = t.allTransactions.session().InteractionRespond(i,
				&discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseUpdateMessage,
					Data: &discordgo.InteractionResponseData{
						Embeds:     []*discordgo.MessageEmbed{embed},
						Components: c,
					},
				})
			if err != nil {
				continue updateLoop
			}
			fields["Info"] = "Queue Updated from interaction"
			break updateLoop
		default:
			_, err = t.allTransactions.session().ChannelMessageEditComplex(
				&discordgo.MessageEdit{
					ID:         queue.MessageID,
					Channel:    queue.ChannelID,
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: c,
				})
			if err != nil {
				t.allTransactions.log.WithFields(log.Fields{
					"ID":      t.id,
					"Type":    t.t,
					"GuildID": t.GuildID(),
				}).Debugf(
					"Transaction queue updating failed: %v",
					err,
				)
				return err
			}
			fields["Info"] = "Queue Updated from message ID"
			break updateLoop
		}
	}
	if !t.quiet {
		t.allTransactions.log.WithFields(fields).Debug(
			"Transaction done",
		)
	}
	return nil
}

func (t *Transactions) addInteraction(i *discordgo.Interaction) {
	t.interactionsSync.Lock()
	defer t.interactionsSync.Unlock()

	go func() {
		time.Sleep(discordgo.InteractionDeadline - (500 * time.Millisecond))
		t.session().InteractionRespond(
			i,
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredMessageUpdate,
			})
	}()

	if _, ok := t.interactions[i.GuildID]; !ok {
		t.interactions[i.GuildID] = make(chan *discordgo.Interaction, 100)
	}
	select {
	case t.interactions[i.GuildID] <- i:
	default:
	}
}
