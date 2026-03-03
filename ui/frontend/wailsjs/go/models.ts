export namespace main {
	
	export class Config {
	    install_dir: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.install_dir = source["install_dir"];
	    }
	}

}

