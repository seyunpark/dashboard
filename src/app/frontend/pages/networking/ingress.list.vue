<template>
	<div class="content-wrapper">
		<div class="content-header">
			<div class="container-fluid">
				<c-navigator group="Networking"></c-navigator>
				<div class="row mb-2">
					<!-- title & search -->
					<div class="col-sm"><h1 class="m-0 text-dark"><span class="badge badge-info mr-2">I</span>Ingresses</h1></div>
					<div class="col-sm-2"><b-form-select v-model="selectedNamespace" :options="namespaces()" size="sm" @input="query_All"></b-form-select></div>
					<div class="col-sm-2 float-left">
						<div class="input-group input-group-sm" >
							<b-form-input id="txtKeyword" v-model="keyword" class="form-control float-right" placeholder="Search"></b-form-input>
							<div class="input-group-append"><button type="submit" class="btn btn-default" @click="query_All"><i class="fas fa-search"></i></button></div>
						</div>
					</div>
					<!-- button -->
					<div class="col-sm-1 text-right">
						<b-button variant="primary" size="sm" @click="$router.push(`/create?context=${currentContext()}&group=Networking&crd=Ingress`)">Create</b-button>
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
								<b-table id="list" hover :items="items" :fields="fields" :filter="keyword" :filter-included-fields="filterOn" @filtered="onFiltered" :current-page="currentPage" :per-page="$config.itemsPerPage" :busy="isBusy" @row-selected="onRowSelected" class="text-sm">
									<template #table-busy>
										<div class="text-center text-success" style="margin:150px 0">
											<b-spinner type="grow" variant="success" class="align-middle mr-2"></b-spinner>
											<span class="align-middle text-lg">Loading...</span>
										</div>
									</template>
									<template v-slot:cell(name)="data">
										<a href="#" @click="sidebar={visible:true, name:data.item.name, src:`${getApiUrl('networking.k8s.io','ingresses',data.item.namespace)}/${data.item.name}`}">{{ data.value }}</a>
									</template>
									<template v-slot:cell(loadBalancers)="data">
										<ul class="list-unstyled mb-0">
											<li v-for="(value, idx) in data.item.loadBalancers" v-bind:key="idx">{{ value[0].ip }}</li>
										</ul>
									</template>
									<template v-slot:cell(rules)="data">
										<ul class="list-unstyled mb-0">
											<li v-for="value in data.item.rules" v-bind:key="value">{{ value }}</li>
										</ul>
									</template>
									<template v-slot:cell(endpoints)="data">
										<ul class="list-unstyled mb-0">
											<li v-for="(d, idx) in data.item.endpoints" v-bind:key="idx">{{ d }}</li>
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
			<c-view crd="Ingress" group="Networking" :name="sidebar.name" :url="sidebar.src" @delete="query_All()" @close="sidebar.visible=false"/>
		</b-sidebar>
	</div>
</template>
<script>
import axios		from "axios"
import VueNavigator from "@/components/navigator"
import VueView from "@/pages/view"
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
				{key: "name", label: "Name", sortable: true},
				{key: "namespace", label: "Namespace", sortable: true},
				{key: "loadBalancers", label: "LoadBalancers"},
				{key: "rules", label: "Rules"},
				{key: "creationTimestamp", label: "Age"},
			],
			isBusy: false,
			items: [],
			currentPage: 1,
			totalItems: 0,
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
	mounted() {
	},
	methods: {
		// 조회
		query_All() {
			this.isBusy = true;
			axios.get(this.getApiUrl("networking.k8s.io","ingresses",this.selectedNamespace))
					.then((resp) => {
						this.items = [];
						resp.data.items.forEach(el => {
							this.items.push({
								name: el.metadata.name,
								namespace: el.metadata.namespace,
								loadBalancers:el.status.loadBalancer ,
								rules: this.getRules(el.spec.rules),
								creationTimestamp: this.$root.getElapsedTime(el.metadata.creationTimestamp),
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
		getRules(p) {
			let list = [];
			for (let i = 0; i<p.length;i++) {
				for (let j = 0; j < p[i].http.paths.length; j++) {
					if (p[i].host !== undefined) {
						list.push(`http://${p[i].host}${p[i].http.paths[j].path} ⇢ ${p[i].http.paths[j].backend["serviceName"]}:${p[i].http.paths[j].backend["servicePort"]}`)
					} else {
						list.push(`http://*${p[i].http.paths[j].path} ⇢ ${p[i].http.paths[j].backend["serviceName"]}:${p[i].http.paths[j].backend["servicePort"]}`);
					}
				}
			}
			return list;
		},
		onRowSelected(items) {
			this.selected = items
			if (this.selected.length !== 0) {
				this.getEvent(this.selected[0].uid)
			}
		},
	},
	beforeDestroy(){
		this.$nuxt.$off('navbar-context-selected')
	}
}
</script>
<style scoped>label {font-weight: 500;}</style>
