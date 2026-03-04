export namespace main {
	
	export class EmbeddingsExportConfig {
	    target_device: string;
	    weight_format: string;
	    extra_quantization_params: string;
	
	    static createFrom(source: any = {}) {
	        return new EmbeddingsExportConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target_device = source["target_device"];
	        this.weight_format = source["weight_format"];
	        this.extra_quantization_params = source["extra_quantization_params"];
	    }
	}
	export class TextGenExportConfig {
	    target_device: string;
	    cache: number;
	    kv_cache_precision: string;
	    enable_prefix_caching: boolean;
	    reasoning_parser: string;
	    tool_parser: string;
	    max_num_batched_tokens: number;
	    max_num_seqs: number;
	
	    static createFrom(source: any = {}) {
	        return new TextGenExportConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target_device = source["target_device"];
	        this.cache = source["cache"];
	        this.kv_cache_precision = source["kv_cache_precision"];
	        this.enable_prefix_caching = source["enable_prefix_caching"];
	        this.reasoning_parser = source["reasoning_parser"];
	        this.tool_parser = source["tool_parser"];
	        this.max_num_batched_tokens = source["max_num_batched_tokens"];
	        this.max_num_seqs = source["max_num_seqs"];
	    }
	}
	export class Config {
	    install_dir: string;
	    uv_url: string;
	    ovms_url: string;
	    startup_set: boolean;
	    search_tags: string[];
	    pipeline_filters: string[];
	    search_limit: number;
	    text_gen_export: TextGenExportConfig;
	    embeddings_export: EmbeddingsExportConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.install_dir = source["install_dir"];
	        this.uv_url = source["uv_url"];
	        this.ovms_url = source["ovms_url"];
	        this.startup_set = source["startup_set"];
	        this.search_tags = source["search_tags"];
	        this.pipeline_filters = source["pipeline_filters"];
	        this.search_limit = source["search_limit"];
	        this.text_gen_export = this.convertValues(source["text_gen_export"], TextGenExportConfig);
	        this.embeddings_export = this.convertValues(source["embeddings_export"], EmbeddingsExportConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class HFModel {
	    id: string;
	    pipeline_tag: string;
	    downloads: number;
	    likes: number;
	    library_name: string;
	
	    static createFrom(source: any = {}) {
	        return new HFModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pipeline_tag = source["pipeline_tag"];
	        this.downloads = source["downloads"];
	        this.likes = source["likes"];
	        this.library_name = source["library_name"];
	    }
	}
	export class StatusResult {
	    uv_ready: boolean;
	    deps_ready: boolean;
	    ovms_ready: boolean;
	    ovms_version: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uv_ready = source["uv_ready"];
	        this.deps_ready = source["deps_ready"];
	        this.ovms_ready = source["ovms_ready"];
	        this.ovms_version = source["ovms_version"];
	    }
	}

}

