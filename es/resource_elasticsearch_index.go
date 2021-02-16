package es

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	elastic7 "github.com/olivere/elastic/v7"
	elastic5 "gopkg.in/olivere/elastic.v5"
	elastic6 "gopkg.in/olivere/elastic.v6"
)

var (
	staticSettingsKeys = []string{
		"number_of_shards",
		"codec",
		"routing_partition_size",
		"load_fixed_bitset_filters_eagerly",
		"shard.check_on_startup",
	}
	dynamicsSettingsKeys = []string{
		"number_of_replicas",
		"auto_expand_replicas",
		"refresh_interval",
		"search.idle.after",
		"max_result_window",
		"max_inner_result_window",
		"max_rescore_window",
		"max_docvalue_fields_search",
		"max_script_fields",
		"max_ngram_diff",
		"max_shingle_diff",
		"blocks_read_only",
		"blocks.read_only_allow_delete",
		"blocks.read",
		"blocks.write",
		"blocks.metadata",
		"max_refresh_listeners",
		"analyze.max_token_count",
		"highlight.max_analyzed_offset",
		"max_terms_count",
		"max_regex_length",
		"routing.allocation.enable",
		"routing.rebalance.enable",
		"gc_deletes",
		"default_pipeline",
		"search.slowlog.threshold.query.warn",
		"search.slowlog.threshold.query.info",
		"search.slowlog.threshold.query.debug",
		"search.slowlog.threshold.query.trace",
		"search.slowlog.threshold.fetch.warn",
		"search.slowlog.threshold.fetch.info",
		"search.slowlog.threshold.fetch.debug",
		"search.slowlog.threshold.fetch.trace",
		"search.slowlog.level",
		"indexing.slowlog.threshold.index.warn",
		"indexing.slowlog.threshold.index.info",
		"indexing.slowlog.threshold.index.debug",
		"indexing.slowlog.threshold.index.trace",
		"indexing.slowlog.level",
		"indexing.slowlog.source",
	}
	settingsKeys = append(staticSettingsKeys, dynamicsSettingsKeys...)
)

var (
	configSchema = map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Description: "Name of the index to create",
			ForceNew:    true,
			Required:    true,
		},
		"force_destroy": {
			Type:        schema.TypeBool,
			Description: "A boolean that indicates that the index should be deleted even if it contains documents.",
			Default:     false,
			Optional:    true,
		},
		// Static settings that can only be set on creation
		"number_of_shards": {
			Type:        schema.TypeString,
			Description: "Number of shards for the index",
			ForceNew:    true,
			Default:     "1",
			Optional:    true,
		},
		"routing_partition_size": {
			Type:        schema.TypeInt,
			Description: "The number of shards a custom routing value can go to. This can be set only on creation.",
			ForceNew:    true,
			Optional:    true,
		},
		"load_fixed_bitset_filters_eagerly": {
			Type:        schema.TypeBool,
			Description: "Indicates whether cached filters are pre-loaded for nested queries.",
			ForceNew:    true,
			Optional:    true,
		},
		"codec": {
			Type:        schema.TypeString,
			Description: "The `default` value compresses stored data with LZ4 compression, but this can be set to `best_compression` which uses DEFLATE for a higher compression ratio.",
			ForceNew:    true,
			Optional:    true,
		},
		"shard_check_on_startup": {
			Type:        schema.TypeString,
			Description: "Whether or not shards should be checked for corruption before opening. When corruption is detected, it will prevent the shard from being opened. Accepts `false`, `true`, `checksum`.",
			ForceNew:    true,
			Optional:    true,
		},
		// Dynamic settings that can be changed at runtime
		"number_of_replicas": {
			Type:        schema.TypeString,
			Description: "Number of shard replicas",
			Optional:    true,
		},
		"auto_expand_replicas": {
			Type:        schema.TypeString, // 0-5 OR 0-all
			Description: "Set the number of replicas to the node count in the cluster",
			Optional:    true,
		},
		"refresh_interval": {
			Type:        schema.TypeString,
			Description: "How often to perform a refresh operation, which makes recent changes to the index visible to search. Can be set to `-1` to disable refresh.",
			Optional:    true,
		},
		"search_idle_after": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"max_result_window": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"max_inner_result_window": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"max_rescore_window": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"max_docvalue_fields_search": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"max_script_fields": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"max_ngram_diff": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"max_shingle_diff": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"max_refresh_listeners": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"analyze_max_token_count": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"highlight_max_analyzed_offset": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"max_terms_count": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"max_regex_length": {
			Type:        schema.TypeInt,
			Description: "",
			Optional:    true,
		},
		"blocks_read_only": {
			Type:        schema.TypeBool,
			Description: "",
			Optional:    true,
		},
		"blocks_read_only_allow_delete": {
			Type:        schema.TypeBool,
			Description: "",
			Optional:    true,
		},
		"blocks_read": {
			Type:        schema.TypeBool,
			Description: "",
			Optional:    true,
		},
		"blocks_write": {
			Type:        schema.TypeBool,
			Description: "",
			Optional:    true,
		},
		"blocks_metadata": {
			Type:        schema.TypeBool,
			Description: "",
			Optional:    true,
		},
		"routing_allocation_enable": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"routing_rebalance_enable": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"gc_deletes": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"default_pipeline": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"search_slowlog_threshold.query.warn": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"search_slowlog_threshold.query.info": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"search_slowlog_threshold_query_debug": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"search_slowlog_threshold_query_trace": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"search_slowlog_threshold_fetch_warn": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"search_slowlog_threshold_fetch_info": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"search_slowlog_threshold_fetch_debug": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"search_slowlog_threshold_fetch_trace": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"search_slowlog_level": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"indexing_slowlog_threshold_index_warn": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"indexing_slowlog_threshold_index_info": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"indexing_slowlog_threshold_index_debug": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"indexing_slowlog_threshold_index_trace": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"indexing_slowlog_level": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		"indexing_slowlog_source": {
			Type:        schema.TypeString,
			Description: "",
			Optional:    true,
		},
		// Other attributes
		"mappings": {
			Type:     schema.TypeString,
			Optional: true,
			// In order to not handle complexities of field mapping updates, updates
			// are not allowed via this provider. See
			// https://www.elastic.co/guide/en/elasticsearch/reference/6.8/indices-put-mapping.html#updating-field-mappings.
			ForceNew:     true,
			ValidateFunc: validation.StringIsJSON,
		},
		"aliases": {
			Type:     schema.TypeString,
			Optional: true,
			// In order to not handle the separate endpoint of alias updates, updates
			// are not allowed via this provider currently.
			ForceNew:     true,
			ValidateFunc: validation.StringIsJSON,
		},
		"rollover_alias": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
	}
)

func resourceElasticsearchIndex() *schema.Resource {
	return &schema.Resource{
		Create: resourceElasticsearchIndexCreate,
		Read:   resourceElasticsearchIndexRead,
		Update: resourceElasticsearchIndexUpdate,
		Delete: resourceElasticsearchIndexDelete,
		Schema: configSchema,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceElasticsearchIndexCreate(d *schema.ResourceData, meta interface{}) error {
	var (
		name     = d.Get("name").(string)
		settings = settingsFromIndexResourceData(d)
		body     = make(map[string]interface{})
		ctx      = context.Background()
		err      error
	)
	if len(settings) > 0 {
		body["settings"] = settings
	}

	if aliasJSON, ok := d.GetOk("aliases"); ok {
		var aliases map[string]interface{}
		bytes := []byte(aliasJSON.(string))
		err = json.Unmarshal(bytes, &aliases)
		if err != nil {
			return fmt.Errorf("fail to unmarshal: %v", err)
		}
		body["aliases"] = aliases
	}

	if mappingsJSON, ok := d.GetOk("mappings"); ok {
		var mappings map[string]interface{}
		bytes := []byte(mappingsJSON.(string))
		err = json.Unmarshal(bytes, &mappings)
		if err != nil {
			return fmt.Errorf("fail to unmarshal: %v", err)
		}
		body["mappings"] = mappings
	}

	// if date math is used, we need to pass the resolved name along to the read
	// so we can pull the right result from the response
	var resolvedName string

	// Note: the CreateIndex call handles URL encoding under the hood to handle
	// non-URL friendly characters and functionality like date math
	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		resp, requestErr := client.CreateIndex(name).BodyJson(body).Do(ctx)
		err = requestErr
		if err == nil {
			resolvedName = resp.Index
		}

	case *elastic6.Client:
		resp, requestErr := client.CreateIndex(name).BodyJson(body).Do(ctx)
		err = requestErr
		if err == nil {
			resolvedName = resp.Index
		}

	default:
		elastic5Client := client.(*elastic5.Client)
		resp, requestErr := elastic5Client.CreateIndex(name).BodyJson(body).Do(ctx)
		err = requestErr
		if err == nil {
			resolvedName = resp.Index
		}

	}

	if err == nil {
		// Let terraform know the resource was created
		d.SetId(resolvedName)
		return resourceElasticsearchIndexRead(d, meta)
	}
	return err
}

func settingsFromIndexResourceData(d *schema.ResourceData) map[string]interface{} {
	settings := make(map[string]interface{})
	for _, key := range settingsKeys {
		schemaName := strings.Replace(key, ".", "_", -1)
		if raw, ok := d.GetOk(schemaName); ok {
			settings[key] = raw
		}
	}
	return settings
}

func indexResourceDataFromSettings(settings map[string]interface{}, d *schema.ResourceData) {
	for _, key := range settingsKeys {
		schemaName := strings.Replace(key, ".", "_", -1)
		err := d.Set(schemaName, settings[key])
		if err != nil {
			log.Printf("[INFO] indexResourceDataFromSettings: %+v", err)
		}
	}
}

func resourceElasticsearchIndexDelete(d *schema.ResourceData, meta interface{}) error {
	var (
		name = d.Id()
		ctx  = context.Background()
		err  error
	)

	if alias, ok := d.GetOk("rollover_alias"); ok {
		name = getWriteIndexByAlias(alias.(string), d, meta)
	}

	// check to see if there are documents in the index
	allowed := allowIndexDestroy(name, d, meta)
	if !allowed {
		return fmt.Errorf("There are documents in the index (or the index could not be , set force_destroy to true to allow destroying.")
	}

	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		_, err = client.DeleteIndex(name).Do(ctx)

	case *elastic6.Client:
		_, err = client.DeleteIndex(name).Do(ctx)

	default:
		elastic5Client := client.(*elastic5.Client)
		_, err = elastic5Client.DeleteIndex(name).Do(ctx)
	}

	return err
}

func allowIndexDestroy(indexName string, d *schema.ResourceData, meta interface{}) bool {
	force := d.Get("force_destroy").(bool)

	var (
		ctx   = context.Background()
		count int64
		err   error
	)
	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return false
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		count, err = client.Count(indexName).Do(ctx)

	case *elastic6.Client:
		count, err = client.Count(indexName).Do(ctx)

	default:
		elastic5Client := client.(*elastic5.Client)
		count, err = elastic5Client.Count(indexName).Do(ctx)
	}

	if err != nil {
		log.Printf("[INFO] allowIndexDestroy: %+v", err)
		return false
	}

	if count > 0 && !force {
		return false
	}
	return true
}

func resourceElasticsearchIndexUpdate(d *schema.ResourceData, meta interface{}) error {
	settings := make(map[string]interface{})
	for _, key := range settingsKeys {
		if d.HasChange(key) {
			settings[key] = d.Get(key)
		}
	}

	// if we're not changing any settings, no-op this function
	if len(settings) == 0 {
		return resourceElasticsearchIndexRead(d, meta)
	}

	body := map[string]interface{}{
		"settings": settings, // should this be index? https://github.com/olivere/elastic/blob/21064b90616b5e9a45df8c406b222d46cad5b14b/indices_put_settings_test.go#L48-L50
	}

	var (
		name = d.Id()
		ctx  = context.Background()
		err  error
	)

	if alias, ok := d.GetOk("rollover_alias"); ok {
		name = getWriteIndexByAlias(alias.(string), d, meta)
	}

	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		_, err = client.IndexPutSettings(name).BodyJson(body).Do(ctx)

	case *elastic6.Client:
		_, err = client.IndexPutSettings(name).BodyJson(body).Do(ctx)

	default:
		elastic5Client := client.(*elastic5.Client)
		_, err = elastic5Client.IndexPutSettings(name).BodyJson(body).Do(ctx)
	}

	if err == nil {
		return resourceElasticsearchIndexRead(d, meta.(*ProviderConf))
	}
	return err
}

func getWriteIndexByAlias(alias string, d *schema.ResourceData, meta interface{}) string {
	var (
		index   = d.Id()
		ctx     = context.Background()
		columns = []string{"index", "is_write_index"}
	)

	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		log.Printf("[INFO] getWriteIndexByAlias: %+v", err)
		return index
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		r, err := client.CatAliases().Alias(alias).Columns(columns...).Do(ctx)
		if err != nil {
			log.Printf("[INFO] getWriteIndexByAlias: %+v", err)
			return index
		}
		for _, column := range r {
			if column.IsWriteIndex == "true" {
				return column.Index
			}
		}

	case *elastic6.Client:
		r, err := client.CatAliases().Alias(alias).Columns(columns...).Do(ctx)
		if err != nil {
			log.Printf("[INFO] getWriteIndexByAlias: %+v", err)
			return index
		}
		for _, column := range r {
			if column.IsWriteIndex == "true" {
				return column.Index
			}
		}

	default:
		elastic5Client := client.(*elastic5.Client)
		r, err := elastic5Client.CatAliases().Alias(alias).Columns(columns...).Do(ctx)
		if err != nil {
			log.Printf("[INFO] getWriteIndexByAlias: %+v", err)
			return index
		}
		for _, column := range r {
			if column.IsWriteIndex == "true" {
				return column.Index
			}
		}
	}

	return index
}

func resourceElasticsearchIndexRead(d *schema.ResourceData, meta interface{}) error {
	var (
		index    = d.Id()
		ctx      = context.Background()
		settings map[string]interface{}
	)

	if alias, ok := d.GetOk("rollover_alias"); ok {
		index = getWriteIndexByAlias(alias.(string), d, meta)
	}

	// The logic is repeated strictly because of the types
	esClient, err := getClient(meta.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		r, err := client.IndexGet(index).Do(ctx)
		if err != nil {
			return err
		}

		if resp, ok := r[index]; ok {
			settings = resp.Settings["index"].(map[string]interface{})
		}
	case *elastic6.Client:
		r, err := client.IndexGet(index).Do(ctx)
		if err != nil {
			return err
		}

		if resp, ok := r[index]; ok {
			settings = resp.Settings["index"].(map[string]interface{})
		}
	default:
		elastic5Client := client.(*elastic5.Client)
		r, err := elastic5Client.IndexGet(index).Do(ctx)
		if err != nil {
			return err
		}

		if resp, ok := r[index]; ok {
			settings = resp.Settings["index"].(map[string]interface{})
		}
	}

	// Don't override name otherwise it will force a replacement
	if _, ok := d.GetOk("name"); !ok {
		name := index
		if providedName, ok := settings["provided_name"].(string); ok {
			name = providedName
		}
		err := d.Set("name", name)
		if err != nil {
			return err
		}
	}

	// If index is managed by ILM or ISM set rollover_alias
	if lifecycle, ok := settings["lifecycle"].(map[string]interface{}); ok {
		if alias, ok := lifecycle["rollover_alias"].(string); ok {
			err := d.Set("rollover_alias", alias)
			if err != nil {
				log.Printf("[INFO] resourceElasticsearchIndexRead: %+v", err)
			}
		}
	} else if opendistro, ok := settings["opendistro"].(map[string]interface{}); ok {
		if ism, ok := opendistro["index_state_management"].(map[string]interface{}); ok {
			if alias, ok := ism["rollover_alias"].(string); ok {
				err := d.Set("rollover_alias", alias)
				if err != nil {
					log.Printf("[INFO] resourceElasticsearchIndexRead: %+v", err)
				}
			}
		}
	}

	indexResourceDataFromSettings(settings, d)

	return nil
}
