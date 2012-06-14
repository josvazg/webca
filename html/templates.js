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

{{define "JSSetupNavigation"}}
	var step=1;
	function next() {
		$('Step'+step).className='shadowed';
		$('form'+step).style.display='none';
		step++;
		helper();
		$('Step'+step).className='activated';
		$('form'+step).style.display='';
	}
	function prev() {
		$('Step'+step).className='shadowed';
		$('form'+step).style.display='none';
		step--;
		helper();
		$('Step'+step).className='activated';
		$('form'+step).style.display='';
	}
	function helper() {
		if(step==1) {
			$('Prev').style.visibility='hidden';
		} else {
			$('Prev').style.visibility='';
		}
		if(step==3) {
			fields=["StreetAddress","PostalCode","Locality","Province",
				    "OrganizationalUnit","Organization","Country"];
			for(i=0;i<fields.length;i++) {
				field=fields[i];
				if($('cert.'+field).value==null || $('cert.'+field).value=='') {
					$('cert.'+field).value=$('ca.'+field).value;
				}
			}
		}
		if(step==4) {
			if($('M.User').value==null || $('M.User').value=='') {
				$('M.User').value=$('Email').value;
			}
			$('Next').style.visibility='hidden';
		} else {
			$('Next').style.visibility='';
		}
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
			$('toggler').innerHTML='{{tr "Less"}}';
		} else {
			tr.style.display='none';
			$('toggler').innerHTML='{{tr "More"}}';
		}
	}
}
{{end}}

{{define "JSCheckpasswd"}}
function showStepError(step, msg) {
	$('noticeText'+step).innerHTML=msg;
	$('notice'+step).style.visibility='';
	$('submit').disabled=true;
}
function hideStepError(currStep) {
	$('notice'+step).style.visibility='hidden';
	$('submit').disabled=false;
}
function checkPassword(currStep,el) {
	if (el.value=="") {
		showStepError(currStep,'{{tr "Type some password!"}}');
		return
	}
	oid=el.id;
	if(oid.substr(-1) === "2") {
		oid=oid.substr(0,oid.length-1);
	} else {
		oid=oid+"2";
	}
	if ($(oid).value!=el.value) {
		showStepError(currStep,'{{tr "Passwords don't match!"}}');
		return
	}
	hideStepError(currStep);
}
{{end}}
