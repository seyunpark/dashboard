<template>
	<div class="content-wrapper">
		<div class="content-header">
			<div class="container-fluid">
				<c-navigator group="Networking"></c-navigator>
				<div class="row mb-2">
					<!-- title & search -->
					<div class="col-sm"><h1 class="m-0 text-dark"><span class="badge badge-info mr-2">S</span>Services</h1></div>
					<div class="col-sm-2"><b-form-select v-model="selectedNamespace" :options="namespaces()" size="sm" @input="query_All"></b-form-select></div>
					<div class="col-sm-2 float-left">
						<div class="input-group input-group-sm" >
							<b-form-input id="txtKeyword" v-model="keyword" class="form-control float-right" placeholder="Search"></b-form-input>
							<div class="input-group-append"><button type="submit" class="btn btn-default" @click="query_All"><i class="fas fa-search"></i></button></div>
						</div>
					</div>
					<!-- button -->
					<div class="col-sm-1 text-right">
						<b-button variant="primary" size="sm" @click="$router.push(`/create?context=${currentContext()}&group=Networking&crd=Service`)">Create</b-button>
					</div>
				</div>
			</div>
		</div>

		<section class="content">
			<div class="container-fluid">
				<!-- count -->
				<div class="row mb-2">
					<div class="col-12 text-right "><span class="text-sm align-middle">Total : {{ totalItems }}</span></div>
				</div>
				<!-- GRID-->
				<div class="row">
					<div class="col-12">
						<div class="card">
							<div class="card-body table-responsive p-0">
								<b-table id="list" hover :items="items" :fields="fields" :filter="keyword" :filter-included-fields="filterOn" @filtered="onFiltered" :current-page="currentPage" :per-page="$config.itemsPerPage" :busy="isBusy" class="text-sm">
									<template #table-busy>
										<div class="text-center text-success" style="margin:150px 0">
											<b-spinner type="grow" variant="success" class="align-middle mr-2"></b-spinner>
											<span class="align-middle text-lg">Loading...</span>
										</div>
									</template>
									<template v-slot:cell(name)="data">
										<a href="#" @click="sidebar={visible:true, name:data.item.name, src:`${getApiUrl('','services',data.item.namespace)}/${data.item.name}`}">{{ data.value }}</a>
									</template>
									<template v-slot:cell(internalEndpoints)="data">
										<ul class="list-unstyled mb-0">
											<li v-for="value in data.item.internalEndpoints" v-bind:key="value">{{ value }}</li>
										</ul>
									</template>
									<template v-slot:cell(externalEndpoints)="data">
										<ul class="list-unstyled mb-0">
											<li v-for="value in data.item.externalEndpoints" v-bind:key="value">{{ value }}</li>
										</ul>
									</template>
									<template v-slot:cell(selector)="data">
										<ul class="list-unstyled mb-0">
											<li v-for="(value, name) in data.item.selector" v-bind:key="name"><span class="badge badge-secondary font-weight-light text-sm">{{ name }}={{ value }}</span></li>
										</ul>
									</template>
									<template v-slot:cell(status)="data">
										<ul class="list-unstyled mb-0">
											<li v-bind:key="data.value" v-bind:class="[ data.value === 'Active'? 'text-success' : 'text-warning' ]">{{ data.value }}</li>
										</ul>
									</template>
								</b-table>
							</div>
							<b-pagination v-model="currentPage" :per-page="$config.itemsPerPage" :total-rows="totalItems" size="sm" align="center"></b-pagination>
						</div>
					</div>
				</div><!-- //GRID-->
			</div>
		</section>
		<b-sidebar v-model="sidebar.visible" width="50em" right shadow no-header>
			<c-view crd="Service" group="Networking" :name="sidebar.name" :url="sidebar.src" @delete="query_All()" @close="sidebar.visible=false"/>
		</b-sidebar>
	</div>
</template>
<script>
import axios		from "axios"
import VueNavigator from "@/components/navigator"
import VueView from "@/pages/view";
export default {
	components: {
		"c-navigator": { extends: VueNavigator },
		"c-view": { extends: VueView }
	},
	data() {
		return {
			selectedNamespace: "",
			keyword: "",
			filterOn: ["name"],
			fields: [
				{ key: "name", label: "Name", sortable: true },
				{ key: "namespace", label: "Namespace", sortable: true },
				{ key: "type", label: "Type", sortable: true },
				{ key: "clusterIP", label: "Cluster IP", sortable: true },
				{ key: "internalEndpoints", label: "Ports", sortable: true },
				{ key: "externalEndpoints", label: "External IP", sortable: true },
				{ key: "selector", label: "Selector", sortable: true },
				{ key: "creationTimestamp", label: "Age", sortable: true },
				{ key: "status", label: "Status", sortable: true }
			],
			isBusy: false,
			items: [],
			status: true,
			currentPage: 1,
			totalItems: 0,
			activeColor: 'green',
			pendingColor: 'orange',
			sidebar: {
				visible: false,
				name: "",
				src: "",
			},
		}
	},
	layout: "default",
	created() {
		this.$nuxt.$on("navbar-context-selected", (ctx) => this.query_All() );
		if(this.currentContext()) this.$nuxt.$emit("navbar-context-selected");
	},
	methods: {
		// 조회
		query_All() {
			this.isBusy = true;
			axios.get(this.getApiUrl("","services",this.selectedNamespace))
					.then((resp) => {
						this.items = [];
						resp.data.items.forEach(el => {
							this.items.push({
								name: el.metadata.name,
								namespace: el.metadata.namespace,
								type: el.spec.type,
								clusterIP: el.spec.clusterIP,
								internalEndpoints: this.toEndpointList(el.spec.ports,el.spec.type),
								externalEndpoints: this.externalEndpoint(el),
								selector: el.spec.selector,
								creationTimestamp: this.$root.getElapsedTime(el.metadata.creationTimestamp),
								status: this.checkStatus(el.spec.type,el.status)
							});
						});
						this.onFiltered(this.items);
					})
					.catch(e => { this.msghttp(e);})
					.finally(()=> { this.isBusy = false;});
		},
		onFiltered(filteredItems) {
			this.totalItems = filteredItems.length;
			this.currentPage = 1
		},
		toEndpointList(p,type) {
			let list = [];
			if (p === undefined) return;
			for(let i =0; i < p.length; i++) {
				if (type === 'NodePort' || type === 'LoadBalancer') {
					list.push(`${p[i].port}:${p[i].nodePort}/${p[i].protocol}`)
				}else if(p[i].targetPort === p[i].port){
					list.push(`${p[i].port}/${p[i].protocol}`)
				}
				else{
					list.push(`${p[i].port}:${p[i].targetPort}/${p[i].protocol}`)
				}
			}
			return list;
		},
		externalEndpoint(el) {
			if (el.spec.type === 'LoadBalancer') {
				return (el.status.loadBalancer.ingress !== undefined? el.status.loadBalancer : "")
			} else {
				return (el.spec.externalIPs? el.spec.externalIPs : "")
			}
		},
		checkStatus(type,status) {
			if (type !== 'LoadBalancer') {
				this.status = true
				return 'Active'
			} else if (this.getExternalIps(status) === true) {
				this.status = true
				return 'Active'
			} else {
				this.status = false
				return 'Pending'
			}
		},
		getExternalIps(status) {
			const lb = this.getLoadBalancer(status);
			return (lb.ingress !== undefined)
		},
		getLoadBalancer(status) {
			return status.loadBalancer
		}
	},
	beforeDestroy(){
		this.$nuxt.$off('navbar-context-selected')
	}
}
</script>
<style scoped>label {font-weight: 500;}</style>
