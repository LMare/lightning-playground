package streamService


import (
	//"github.com/google/uuid"
	"sync"
	"io"
	"fmt"
	"reflect"
	"encoding/json"
	//"time"
	lnrpc "github.com/Lmare/lightning-test/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"


	//exception "github.com/Lmare/lightning-test/backend/exception"
)
/*
type Stream interface {
    Recv() (any, error)
    Close() error
}*/

// gereric structure for the stream
type StreamWrapper[T any] struct {
    RecvCallback func() (*T, error)
    CloseCallback func() error
}
func (s StreamWrapper[T]) Recv() (any, error) {
    return s.RecvCallback()
}

func (s StreamWrapper[T]) Close() error {
    return s.CloseCallback()
}


// save server stream
// map[string]StreamWrapper
// TODO have une struct Envelop to have a batch garbage Collector in case
var sessionChannelMap = sync.Map{}


/** TODO: GESTION Multi-Onglet
	comment structurer ça correctement dans mon code ?
	J'ai une map qui pour le moment associe un canal à un utilistateur pour centraliser la production des notifications (je ferais surement un petit wrapper plus tart pour gérer différents types d'event, faut pas monter trop rapidement en complexité ^^')
	j'ai une map de liste de connexion SSE (http.ResponseWriter)
	J'ai un handler qui à la création ou détruction de la requête souscrit ou revoque l'abonnement au évenement.
	Et du coup il me faut plus qu'une go routine (créé à l'initialisation de la session) qui permet de de flush dans les ResponseWriter dès que des messages arrives dans le canal.

	🛠️ Points d’attention
	- Utilise des canaux bufferisés (make(chan Event, N)) pour éviter de bloquer les producteurs si le consommateur est lent.
	- Ajoute un ping/keep‑alive régulier pour maintenir la connexion ouverte (et éviter que des proxies la coupent).
	- Surveille la taille des listes de clients pour éviter les fuites mémoire si un utilisateur ouvre/ferme beaucoup d’onglets.



	donc du coup avec cette configuration là j'ai les notifications sur tous les onglets.
	je me suis dit que si je veux des nofitications qui s'affiche uniquement sur certain onglets je peux faire ça : (note j'utilise HTMX, mais on pourrait avoir plus ou moin la même logique en rest classique)
	dans le fait une action qui produit un stream gRPC, je génère un uuid que je met dans mon StreamWrapper,
	je retourne au navigateur du html qui defini une class css qui dépends de cett uuid qui fait un display block.
	dans les event SSE je génère une envelopper HTML sur le message qui ajoute une classe pour mettre les notif mono-onglet en display none + la classe unique qui permet d'afficher seulement dans l'onglet qui contient la définition.
*/



func GetChannel(sessionId string) chan string{
	channel, ok := sessionChannelMap.Load(sessionId)
	if !ok {
		fmt.Println("initialisation du chanel")
		channel = make(chan string)
		sessionChannelMap.Store(sessionId, channel)
	}
	return channel.(chan string)
}

// save the steam in context of the server
func StreamResult[T any](stream StreamWrapper[T]) {
	//id := uuid.New().String()
	id := "uniqueSession"
	channel := GetChannel(id)
	go func() {
        for {
            msg, err := stream.Recv()
			if err == io.EOF {
				fmt.Println("fin de la goRoutine")
				break // stream terminé
			} else if err != nil {
				fmt.Println("Erreur sur le stream", err)
                channel <- fmt.Sprintf("Erreur : %s", err)
				break
            } else {
				fmt.Println("Data", msg)
	            channel <- encode(msg)
			}
        }
    }()
}

// ------

// encode transforme n'importe quelle valeur en string pour SSE
func encode(v interface{}) string {
    switch val := v.(type) {
    case string:
        return val
	case *lnrpc.Payment :
		return fmt.Sprintf("💸 Paiement de %d sats — statut : %s", val.ValueSat, val.Status.String())
    case fmt.Stringer:
        return val.String()
    default:
        // Si c'est un type simple (int, float, bool, etc.)
        rv := reflect.ValueOf(v)
        switch rv.Kind() {
        case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
            reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
            reflect.Float32, reflect.Float64, reflect.Bool:
            return fmt.Sprintf("%v", v)
        default:
            // Pour les structs, slices, maps, etc. → JSON
            jsonData, err := json.Marshal(v)
            if err != nil {
                return fmt.Sprintf("error: %v", err)
            }
            return string(jsonData)
        }
    }
}
