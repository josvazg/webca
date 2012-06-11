{{define "JSGetID"}}
function $(id) {
	return document.getElementById(id);
}
{{end}}

{{define "JSByClass"}}
function getElementsByClass( searchClass, domNode, tagName) { 
	if (domNode == null) domNode = document;
	if (tagName == null) tagName = '*';
	var el = new Array();
	var tags = domNode.getElementsByTagName(tagName);
	var tcl = " "+searchClass+" ";
	for(i=0,j=0; i<tags.length; i++) { 
		var test = " " + tags[i].className + " ";
		if (test.indexOf(tcl) != -1) 
			el[j++] = tags[i];
	} 
	return el;
} 
{{end}}

{{define "JSNavigation"}}
	var step=1;
	var transitions=1;
	function next() {
		$('Step'+step).className='shadowed';
		$('form'+step).style.display='none';
		step++;
		transitions++;
		$('Step'+step).className='activated';
		$('form'+step).style.display='';
	}
	function prev() {
		$('Step'+step).className='shadowed';
		$('form'+step).style.display='none';
		step--;
		$('Step'+step).className='activated';
		$('form'+step).style.display='';
	}
{{end}}

{{define "JSToggleOps"}}
{{template "JSByClass"}}
function toggleOps(el) {
	trs=getElementsByClass("ops",el,"tr");
	for (i=0;i<trs.length;i++) {
		tr=trs[i]
		if (tr.style.display=='none') {
			tr.style.display='';
		} else {
			tr.style.display='none';
		}
	}
}
{{end}}
