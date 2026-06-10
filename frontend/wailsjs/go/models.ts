export namespace main {
	
	export class EnvConfig {
	    Java: any[];
	    Python: string;
	
	    static createFrom(source: any = {}) {
	        return new EnvConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Java = source["Java"];
	        this.Python = source["Python"];
	    }
	}
	export class ToolConfig {
	    ID: string;
	    Name: string;
	    Type: string;
	    Path: string;
	    JavaVersion: string;
	    Description: string;
	    Category: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.Type = source["Type"];
	        this.Path = source["Path"];
	        this.JavaVersion = source["JavaVersion"];
	        this.Description = source["Description"];
	        this.Category = source["Category"];
	    }
	}

}

