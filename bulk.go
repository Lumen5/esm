/*
Copyright 2016 Medcl (m AT medcl.net)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"encoding/json"
	"github.com/cheggaaa/pb"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
)

func (c *Migrator) NewBulkWorker(docCount *int, pb *pb.ProgressBar, wg *sync.WaitGroup) {

	log.Debug("start es bulk worker")

	bulkItemSize := 0
	mainBuf := bytes.Buffer{}
	docBuf := bytes.Buffer{}
	docEnc := json.NewEncoder(&docBuf)

	idleDuration := 5 * time.Second
	idleTimeout := time.NewTimer(idleDuration)
	defer idleTimeout.Stop()

	taskTimeOutDuration := 5 * time.Minute
	taskTimeout := time.NewTimer(taskTimeOutDuration)
	defer taskTimeout.Stop()


READ_DOCS:
	for {
		idleTimeout.Reset(idleDuration)
		taskTimeout.Reset(taskTimeOutDuration)
		select {
		case docI, open := <-c.DocChan:
			var err error

			// if channel is closed, flush and exit
			if !open {
				log.Debug("DocChan closed, finishing bulk worker")
				goto WORKER_DONE
			}

			log.Trace("read doc from channel,", docI)
			// this check is in case the document is an error with scroll stuff
			if status, ok := docI["status"]; ok {
				if status.(int) == 404 {
					log.Error("error: ", docI["response"])
					continue
				}
			}

			// sanity check (ES 8.x does not require _type)
			// If a document is missing _index, _source, or _id, the scroll is corrupted
			for _, key := range []string{"_index", "_source", "_id"} {
				if _, ok := docI[key]; !ok {
					log.Error("FATAL: Document missing required field: ", key)
					log.Error("This indicates the scroll response is corrupted or malformed")
					log.Error("Document received:", docI)
					panic("Scroll returned malformed document - aborting to prevent data loss")
				}
			}

			var tempDestIndexName string
			var tempTargetTypeName string
			tempDestIndexName = docI["_index"].(string)
			// ES 8.x does not have _type, default to _doc
			if typeVal, ok := docI["_type"]; ok {
				tempTargetTypeName = typeVal.(string)
			} else {
				tempTargetTypeName = "_doc" // ES 7+/8+ default
			}

			if c.Config.TargetIndexName != "" {
				tempDestIndexName = c.Config.TargetIndexName
			}

			if c.Config.OverrideTypeName != "" {
				tempTargetTypeName = c.Config.OverrideTypeName
			}

			doc := Document{
				Index:  tempDestIndexName,
				Type:   tempTargetTypeName,
				source: docI["_source"].(map[string]interface{}),
				Id:     docI["_id"].(string),
			}

			if c.Config.RegenerateID {
				doc.Id = ""
			}

			if c.Config.RenameFields != "" {
				kvs := strings.Split(c.Config.RenameFields, ",")
				for _, i := range kvs {
					fvs := strings.Split(i, ":")
					oldField := strings.TrimSpace(fvs[0])
					newField := strings.TrimSpace(fvs[1])
					if oldField == "_type" {
						doc.source[newField] = docI["_type"].(string)
					} else {
						v := doc.source[oldField]
						doc.source[newField] = v
						delete(doc.source, oldField)
					}
				}
			}

			// add doc "_routing" if exists
			if _, ok := docI["_routing"]; ok {
				str, ok := docI["_routing"].(string)
				if ok && str != "" {
					doc.Routing = str
				}
			}

			// sanity check
			if len(doc.Index) == 0 || len(doc.Type) == 0 {
				log.Errorf("failed decoding document: %+v", doc)
				continue
			}

			// encode the doc and and the _source field for a bulk request
			// For ES 8.x, omit _type from bulk metadata
			metadata := map[string]interface{}{
				"_index": doc.Index,
				"_id":    doc.Id,
			}
			if doc.Routing != "" {
				metadata["routing"] = doc.Routing
			}
			post := map[string]interface{}{
				"index": metadata,
			}
			if err = docEnc.Encode(post); err != nil {
				log.Error(err)
			}
			if err = docEnc.Encode(doc.source); err != nil {
				log.Error(err)
			}


			// append the doc to the main buffer
			mainBuf.Write(docBuf.Bytes())
			// reset for next document
			bulkItemSize++
			(*docCount)++
			docBuf.Reset()

			// if we approach the 100mb es limit, flush to es and reset mainBuf
			if mainBuf.Len()+docBuf.Len() > (c.Config.BulkSizeInMB * 1024*1024) {
				goto CLEAN_BUFFER
			}

		case <-idleTimeout.C:
			log.Debug("5s no message input")
			goto CLEAN_BUFFER
		case <-taskTimeout.C:
			log.Warn("5m no message input, close worker")
			goto WORKER_DONE
		}

		goto READ_DOCS

	CLEAN_BUFFER:
		c.TargetESAPI.Bulk(&mainBuf)
		log.Trace("clean buffer, and execute bulk insert")
		pb.Add(bulkItemSize)
		bulkItemSize = 0
		if c.Config.SleepSecondsAfterEachBulk >0{
			time.Sleep(time.Duration(c.Config.SleepSecondsAfterEachBulk) * time.Second)
		}
	}
WORKER_DONE:
	if docBuf.Len() > 0 {
		mainBuf.Write(docBuf.Bytes())
		bulkItemSize++
	}
	c.TargetESAPI.Bulk(&mainBuf)
	log.Trace("bulk insert")
	pb.Add(bulkItemSize)
	bulkItemSize = 0
	wg.Done()
}
