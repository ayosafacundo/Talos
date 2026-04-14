export namespace main {
	
	export class AppManifestView {
	    id: string;
	    name: string;
	    icon: string;
	    url: string;
	    description: string;
	    category: string;
	    store_url?: string;
	    allowed_origins?: string[];
	    development?: boolean;
	    trust_status?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppManifestView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.url = source["url"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.store_url = source["store_url"];
	        this.allowed_origins = source["allowed_origins"];
	        this.development = source["development"];
	        this.trust_status = source["trust_status"];
	    }
	}
	export class PermissionAuditEntry {
	    ts: string;
	    action: string;
	    app_id: string;
	    scope: string;
	    granted: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new PermissionAuditEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ts = source["ts"];
	        this.action = source["action"];
	        this.app_id = source["app_id"];
	        this.scope = source["scope"];
	        this.granted = source["granted"];
	        this.message = source["message"];
	    }
	}
	export class PermissionEntry {
	    app_id: string;
	    scope: string;
	    granted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PermissionEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app_id = source["app_id"];
	        this.scope = source["scope"];
	        this.granted = source["granted"];
	    }
	}
	export class RemotePackageDescriptor {
	    id: string;
	    name: string;
	    source?: string;
	    install_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new RemotePackageDescriptor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.source = source["source"];
	        this.install_url = source["install_url"];
	    }
	}
	export class ThemeInfo {
	    name: string;
	    file: string;
	
	    static createFrom(source: any = {}) {
	        return new ThemeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.file = source["file"];
	    }
	}
	export class UpdateEntryView {
	    app_id: string;
	    version: string;
	    artifact_url: string;
	    min_host_version?: string;
	    signature_url?: string;
	    name?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateEntryView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app_id = source["app_id"];
	        this.version = source["version"];
	        this.artifact_url = source["artifact_url"];
	        this.min_host_version = source["min_host_version"];
	        this.signature_url = source["signature_url"];
	        this.name = source["name"];
	    }
	}
	export class UserPrefs {
	    theme: string;
	    tab_colors: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new UserPrefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.tab_colors = source["tab_colors"];
	    }
	}

}

